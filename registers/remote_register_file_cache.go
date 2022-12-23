package registers

import (
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"os"
)

type RemoteRegisterFileCache struct {
	blockHeight uint64
	registers   map[RegisterKey]flow.RegisterValue

	log zerolog.Logger
}

var _ RegisterGetWrapper = &RemoteRegisterFileCache{}

func NewRemoteRegisterFileCache(
	blockHeight uint64,
	log zerolog.Logger,
) (*RemoteRegisterFileCache, error) {
	c := &RemoteRegisterFileCache{
		blockHeight: blockHeight,
		log:         log,
		registers:   make(map[RegisterKey]flow.RegisterValue),
	}
	err := c.open()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RemoteRegisterFileCache) Wrap(registerFunc RegisterGetRegisterFunc) RegisterGetRegisterFunc {
	return func(owner string, key string) (flow.RegisterValue, error) {
		val, found := c.registers[RegisterKey{owner, key}]
		if found {
			return val, nil
		}
		val, err := registerFunc(owner, key)
		if err != nil {
			return nil, err
		}
		c.registers[RegisterKey{owner, key}] = val
		return val, nil
	}
}

// Close the cache
func (c *RemoteRegisterFileCache) Close() error {
	// overwrite existing file
	// and dump registers to file as a csv
	filename := c.getFilename()
	csvFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err := csvFile.Close()
		if err != nil {
			c.log.Error().Err(err).Msg("error closing csv file")
		}
	}()
	c.log.
		Info().
		Int("registers", len(c.registers)).
		Msgf("closing cache file: %s", filename)

	csvwriter := csv.NewWriter(csvFile)
	defer csvwriter.Flush()
	for key, val := range c.registers {
		encodedKey := key.ToReadable()
		encodedValue := c.encodeRegisterValue(val)
		err := csvwriter.Write([]string{encodedKey.Owner, encodedKey.Key, encodedValue})
		if err != nil {
			return err
		}
	}
	return nil
}

// open opens the cache by loading registers from a file
func (c *RemoteRegisterFileCache) open() error {
	filename := c.getFilename()

	c.log.Info().Msgf("opening cache file: %s", filename)

	csvFile, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// file does not exist
			c.log.Info().Msgf("cache file does not exist: %s", filename)
			return nil
		}
		return err
	}
	defer func() { _ = csvFile.Close() }()

	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		return err
	}
	for _, line := range csvLines {
		if len(line) != 3 {
			return fmt.Errorf("invalid line: %v", line)
		}
		owner := line[0]
		key := line[1]
		value := line[2]
		decodedValue, err := c.decodeRegisterValue(value)
		if err != nil {
			return err
		}
		decodedKey := RegisterKey{owner, key}.ToMangled()
		c.registers[decodedKey] = decodedValue
	}

	return nil
}

// encode register value
func (c *RemoteRegisterFileCache) encodeRegisterValue(value flow.RegisterValue) string {
	return hex.EncodeToString(value)
}

// decode register value
func (c *RemoteRegisterFileCache) decodeRegisterValue(value string) (flow.RegisterValue, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

// getFilename
func (c *RemoteRegisterFileCache) getFilename() string {
	return fmt.Sprintf("block-%d-cache.csv", c.blockHeight)
}
