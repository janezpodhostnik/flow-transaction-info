package registers

import (
	"encoding/csv"
	"fmt"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"strconv"
)

type registerReadEntry struct {
	key  RegisterKey
	read int
}

func (e registerReadEntry) String() string {
	return fmt.Sprintf("%v: %v bytes", e.key, e.read)
}

type RemoteRegisterReadTracker struct {
	registerRead []registerReadEntry
	filename     string

	log zerolog.Logger
}

var _ RegisterGetWrapper = &RemoteRegisterReadTracker{}

func NewRemoteRegisterReadTracker(directory string, log zerolog.Logger) *RemoteRegisterReadTracker {
	return &RemoteRegisterReadTracker{
		filename:     directory + "/registers_read.csv",
		registerRead: []registerReadEntry{},
		log:          log,
	}
}

func (r *RemoteRegisterReadTracker) Wrap(inner RegisterGetRegisterFunc) RegisterGetRegisterFunc {
	return func(owner string, key string) (flow.RegisterValue, error) {
		val, err := inner(owner, key)
		k := RegisterKey{owner, key}.ToReadable()

		if err != nil {
			return nil, err
		}

		r.registerRead = append(r.registerRead, registerReadEntry{
			key:  k,
			read: len(val),
		})

		return val, nil
	}
}

func (r *RemoteRegisterReadTracker) Close() error {
	err := os.MkdirAll(filepath.Dir(r.filename), os.ModePerm)
	if err != nil {
		return err
	}

	csvFile, err := os.Create(r.filename)
	if err != nil {
		return err
	}
	defer func() {
		err := csvFile.Close()
		if err != nil {
			r.log.Error().Err(err).Msg("error closing csv file")
		}
	}()

	csvwriter := csv.NewWriter(csvFile)
	defer csvwriter.Flush()
	err = csvwriter.Write([]string{"# Sequence", "Owner", "Key", "bytes"})
	if err != nil {
		return err
	}
	for n, read := range r.registerRead {

		err := csvwriter.Write([]string{strconv.Itoa(n + 1), read.key.Owner, read.key.Key, strconv.Itoa(read.read)})
		if err != nil {
			return err
		}
	}
	return nil
}
