package local

import (
	"encoding/base64"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/boltdb/bolt"
	"github.com/deixis/spine/crypto"
	pb "github.com/deixis/spine/schedule/adapter/local/localpb"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	dbFileMod   = 0600
	partitionBy = int64(time.Hour)
)

var (
	// ErrMarshalling occurs when a storage message cannot be marshalled
	ErrMarshalling = errors.New("schedule marshalling error")
	// ErrUnmarshalling occurs when a storage message cannot be unmarshalled
	ErrUnmarshalling = errors.New("schedule unmarshalling error")

	eventBucket       = []byte("event")
	partitionBucket   = []byte("partition")
	checkpointBuckets = []byte("checkpoint")
	bucketKeys        = [][]byte{
		eventBucket,
		partitionBucket,
		checkpointBuckets,
	}

	lastCheckpointKey = []byte("last")

	errDatabaseClosed = errors.New("db closed")
)

type storage struct {
	state  uint32
	db     *bolt.DB
	crypto *crypto.Rotor

	checkpoint int64
}

func newStorage(c *EncryptionConfig) (*storage, error) {
	storage := &storage{}
	if c != nil {
		keys := make(map[uint32][]byte)
		for i, key := range c.Keys {
			decodedKey, err := base64.StdEncoding.DecodeString(key)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot decode storage key #%d", i)
			}
			if len(decodedKey) != crypto.KeySize {
				return nil, errors.Errorf(
					"invalid encryption key length %d != %d", len(decodedKey), crypto.KeySize,
				)
			}
			keys[uint32(i)] = decodedKey
		}
		storage.crypto = crypto.NewRotor(keys, c.Default)
	}
	return storage, nil
}

// Open opens the database. This function must be called first.
func (s *storage) Open(path string) error {
	db, err := bolt.Open(
		path,
		dbFileMod,
		&bolt.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		return errors.Wrap(err, "error opening schedule database")
	}

	// Create buckets if needed
	err = db.Update(func(tx *bolt.Tx) error {
		for _, bk := range bucketKeys {
			_, err := tx.CreateBucketIfNotExists(bk)
			if err != nil {
				return errors.Wrapf(err, "error creating/loading bucket <%s>", bk)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	atomic.StoreUint32(&s.state, 1)
	s.db = db

	s.checkpoint = s.loadLastCheckpoint()
	return nil
}

// Save persists e to the storage
func (s *storage) Save(e *pb.Event) error {
	if atomic.LoadUint32(&s.state) == 0 {
		return errDatabaseClosed
	}

	e.Id = e.Job.Id + "/" + strconv.FormatUint(uint64(e.Attempt), 10)

	return s.db.Batch(func(tx *bolt.Tx) error {
		if e.Due < s.checkpoint {
			// Postpone event to make sure it will be executed
			e.Due = s.checkpoint
		}
		evtData, err := s.marshal(e)
		if err != nil {
			return ErrMarshalling
		}

		partKey := partitionKey(e.Due)
		eventKey := eventKey(e)

		parts := tx.Bucket(partitionBucket)
		events := tx.Bucket(eventBucket)

		// Add event to partition
		part := pb.Partition{}
		partData := parts.Get(partKey)
		if len(partData) > 0 {
			if err := s.unmarshal(partData, &part); err != nil {
				return ErrUnmarshalling
			}
		}
		if part.From == 0 && part.To == 0 {
			part.From = partitionStart(e.Due)
			part.To = partitionEnd(e.Due)
		}
		part.Keys = append(part.Keys, string(eventKey))
		sort.Strings(part.Keys)
		partData, err = s.marshal(&part)
		if err != nil {
			return ErrMarshalling
		}

		if err := events.Put(eventKey, evtData); err != nil {
			return errors.Wrap(err, "error creating event record")
		}
		if err := parts.Put(partKey, partData); err != nil {
			return errors.Wrap(err, "error updating index record")
		}
		return nil
	})
}

// Load loads events due from the last checkpoint to t
func (s *storage) Load(t int64) (l []*pb.Event, next int64, err error) {
	if atomic.LoadUint32(&s.state) == 0 {
		return nil, 0, errDatabaseClosed
	}

	return l, next, s.db.Batch(func(tx *bolt.Tx) error {
		from := s.checkpoint
		to := t
		s.checkpoint = t + 1

		next = partitionEnd(t) + 1

		parts := tx.Bucket(partitionBucket)
		events := tx.Bucket(eventBucket)
		checkpoints := tx.Bucket(checkpointBuckets)

		for t := partitionStart(from); t <= partitionStart(to); t += partitionBy {
			partKey := partitionKey(t)
			part := pb.Partition{}
			partData := parts.Get(partKey)
			if len(partData) > 0 {
				if err := s.unmarshal(partData, &part); err != nil {
					return ErrUnmarshalling
				}
			}

			for _, key := range part.Keys {
				e := pb.Event{}
				if err := s.unmarshal(events.Get([]byte(key)), &e); err != nil {
					return ErrUnmarshalling
				}
				if from <= e.Due && e.Due <= to {
					l = append(l, &e)
				}
				if to < e.Due && e.Due < next {
					next = e.Due
				}
			}
		}

		seq, err := checkpoints.NextSequence()
		if err != nil {
			return err
		}
		checkpointData, err := s.marshal(&pb.Checkpoint{
			Seq:  seq,
			From: from,
			To:   to,
		})
		if err != nil {
			return ErrMarshalling
		}
		if err := checkpoints.Put(lastCheckpointKey, checkpointData); err != nil {
			return errors.Wrap(err, "error creating log record")
		}
		return nil
	})
}

func (s *storage) loadLastCheckpoint() (t int64) {
	// Default value to make sure old events won't be re-processed
	t = time.Now().UnixNano()

	s.db.Batch(func(tx *bolt.Tx) error {
		checkpoints := tx.Bucket(checkpointBuckets)

		data := checkpoints.Get(lastCheckpointKey)
		if len(data) == 0 {
			return nil
		}

		cp := pb.Checkpoint{}
		if err := s.unmarshal(data, &cp); err != nil {
			return ErrUnmarshalling
		}
		if cp.Seq == checkpoints.Sequence() {
			t = cp.To
		}
		return nil
	})
	return t
}

// Close implements io.Closer
func (s *storage) Close() error {
	if s.db == nil {
		return nil
	}
	atomic.StoreUint32(&s.state, 0)
	return s.db.Close()
}

func (s *storage) marshal(pb proto.Message) ([]byte, error) {
	if s.crypto == nil {
		return proto.Marshal(pb)
	}

	plain, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return s.crypto.Encrypt(plain)
}

func (s *storage) unmarshal(buf []byte, pb proto.Message) error {
	if s.crypto == nil {
		return proto.Unmarshal(buf, pb)
	}

	plain, err := s.crypto.Decrypt(buf)
	if err != nil {
		return err
	}
	return proto.Unmarshal(plain, pb)
}

func eventKey(e *pb.Event) []byte {
	return []byte(strings.Join([]string{
		strconv.FormatInt(e.Due, 10),
		e.Id,
	}, "/"))
}

func partitionKey(t int64) []byte {
	return []byte(strconv.FormatInt(partitionStart(t), 10))
}

func partitionStart(t int64) int64 {
	return t - (abs(t) % partitionBy)
}

func partitionEnd(t int64) int64 {
	to := partitionStart(t) + partitionBy - 1
	return to
}

func abs(i int64) int64 {
	if i < 0 {
		return i * -1
	}
	return i
}
