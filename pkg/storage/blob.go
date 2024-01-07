package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/golang/snappy"
	"github.com/google/uuid"
)

type BlobStoreConfig struct {
	// BaseDir - base directory where blobs will be stored
	BaseDir string

	Compress bool

	// Host - if true, it will host the api
	Host bool
	// HostAddr - address where it hosts api
	HostAddr string
}

type DiskBlobStore struct {
	router *chi.Mux
	config *BlobStoreConfig
	kv     *badger.DB
}

func NewDiskBlobStore(opts BlobStoreConfig) *DiskBlobStore {
	return &DiskBlobStore{
		config: &opts,
	}
}

// creates base dir, opens kv db, creates grpc server
func (b *DiskBlobStore) Init() error {
	// create the base dir
	if err := b.createDirs(); err != nil {
		return err
	}

	// Open the KV store
	kv, err := badger.Open(badger.DefaultOptions(b.config.BaseDir + "/kv.db"))
	if err != nil {
		return err
	}

	b.kv = kv

	// init router
	b.router = chi.NewRouter()

	// register handlers
	b.router.Get("/blob/{deployID}", makeAPIHandler(b.handleGetBlob))
	b.router.Post("/blob/{deployID}", makeAPIHandler(b.handleCreateBlob))
	b.router.Delete("/blob/{deployID}", makeAPIHandler(b.handleDeleteBlob))
	b.router.Get("/stats", makeAPIHandler(b.handleStats))

	if b.config.HostAddr == "" {
		b.config.HostAddr = "127.0.0.1:3069"
	}

	if b.config.Host {
		go func() {
			slog.Info("hosting blob api", "addr", b.config.HostAddr)
			log.Fatal(http.ListenAndServe(b.config.HostAddr, b.router))
		}()
	}

	return nil
}

func (b *DiskBlobStore) createDirs() error {
	return os.MkdirAll(b.config.BaseDir, 0755)
}

func (b *DiskBlobStore) GetPath(s string) string {

	return fmt.Sprintf("%s/%s/%s/%s/%s.bin", b.config.BaseDir, s[:1], s[:2], s[:3], s)
}

func (b *DiskBlobStore) CreateBlob(id uuid.UUID, data []byte) error {
	path := b.GetPath(id.String())

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if b.config.Compress {
		st := time.Now()
		data, err = b.compress(data)
		if err != nil {
			return err
		}
		fmt.Printf("compress took: %s\n", time.Since(st))
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	// once written add it to kv
	// id : size of blob
	err = b.kv.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(id.String()), []byte(strconv.Itoa(len(data))))
		return err
	})

	return err
}

func (b *DiskBlobStore) GetBlob(id uuid.UUID) ([]byte, error) {
	exists := false
	// check if id exists in kv
	_ = b.kv.View(func(txn *badger.Txn) error {
		d, _ := txn.Get([]byte(id.String()))
		if d != nil {
			exists = true
		}
		return nil
	})

	if !exists {
		return nil, fmt.Errorf("blob with id %s does not exist", id.String())
	}

	f, err := os.Open(b.GetPath(id.String()))
	if err != nil {
		return nil, err
	}

	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if !b.config.Compress {
		return bytes, nil
	}

	st := time.Now()
	decompressed, err := b.decompress(bytes)
	if err != nil {
		return nil, err
	}
	fmt.Printf("decompress took: %s\n", time.Since(st))

	return decompressed, nil
}

func (b *DiskBlobStore) DeleteBlob(id uuid.UUID) error {
	err := os.Remove(b.GetPath(id.String()))
	if err != nil {
		return err
	}

	// delete from kv
	err = b.kv.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(id.String()))
		return err
	})

	return err
}

func (b *DiskBlobStore) GetNumberOfBlobs() (int, error) {
	// get all keys from kv

	count := 0
	err := b.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.PrefetchSize = 0
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}

		return nil
	})

	return count, err
}

func (b *DiskBlobStore) GetSizeFromDisk() (int64, error) {
	// get size of base dir
	var size int64
	err := filepath.Walk(b.config.BaseDir, func(path string, info os.FileInfo, err error) error {
		// if kv.db in path, skip
		if strings.Contains(path, "kv.db") {
			return nil
		}

		size += info.Size()
		return nil
	})

	return size, err
}

func (b *DiskBlobStore) GetSizeFromKV() (int64, error) {
	var size int64
	err := b.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			it.Item().Value(func(val []byte) error {
				s, _ := strconv.Atoi(string(val))
				size += int64(s)
				return nil
			})
		}

		return nil
	})

	return size, err
}

func (b *DiskBlobStore) compress(data []byte) ([]byte, error) {
	return snappy.Encode(nil, data), nil
}

func (b *DiskBlobStore) decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}

func (b *DiskBlobStore) Close() error {
	return b.kv.Close()
}

// api handlers
func (b *DiskBlobStore) handleGetBlob(w http.ResponseWriter, r *http.Request) error {
	deployID := chi.URLParam(r, "deployID")
	fmt.Println(deployID)

	id, err := uuid.Parse(deployID)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	bytes, err := b.GetBlob(id)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(bytes)
	return nil
}

func (b *DiskBlobStore) handleCreateBlob(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(chi.URLParam(r, "deployID"))
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	err = b.CreateBlob(id, bytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	return writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (b *DiskBlobStore) handleDeleteBlob(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(chi.URLParam(r, "deployID"))
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	err = b.DeleteBlob(id)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	return writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func (b *DiskBlobStore) handleStats(w http.ResponseWriter, r *http.Request) error {
	type stats struct {
		NumberOfBlobs  int64   `json:"number_of_blobs"`
		SizeFromDisk   int64   `json:"size_from_disk"`
		SizeFromDiskMb float64 `json:"size_from_disk_mb"`
		SizeFromKV     int64   `json:"size_from_kv"`
		SizeFromKVMb   float64 `json:"size_from_kv_mb"`
	}

	s := stats{}

	n, err := b.GetNumberOfBlobs()
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	s.NumberOfBlobs = int64(n)

	size, err := b.GetSizeFromDisk()
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	s.SizeFromDisk = size
	s.SizeFromDiskMb = float64(size) / 1024 / 1024

	size, err = b.GetSizeFromKV()
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, errorResponse(err))
	}

	s.SizeFromKV = size
	s.SizeFromKVMb = float64(size) / 1024 / 1024

	return writeJSON(w, http.StatusOK, s)
}

func errorResponse(err error) map[string]string {
	return map[string]string{"error": err.Error()}
}

type apiHandler func(w http.ResponseWriter, r *http.Request) error

func makeAPIHandler(h apiHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			// todo
			slog.Error("api handler error", "err", err)
		}
	}
}
