package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/decred/dcrlnlpd/rpc"
)

const (
	invoicesDir = "invoices"
)

var (
	errNotExists = errors.New("does not exist")
)

type waitingInvoice struct {
	TargetNode  rpc.NodeID `json:"target_node"`
	ChannelSize uint64     `json:"channel_size"`
}

// writeJsonFile writes the given data as a JSON file.
func (s *Server) writeJsonFile(fpath string, data interface{}) error {
	dir := filepath.Dir(fpath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.Create(fpath)
	if err != nil {
		return err
	}

	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(data)
}

// readJsonFile reads the json data into the var.
func (s *Server) readJsonFile(fpath string, data interface{}) error {
	f, err := os.Open(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return errNotExists
		}
		return err
	}

	defer f.Close()
	dec := json.NewDecoder(f)
	return dec.Decode(data)
}

// removeFile removes the specified file. This doesn't return an error if the
// file does not exist.
func (s *Server) removeFile(fpath string) error {
	err := os.Remove(fpath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
