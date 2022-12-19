package main

import (
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"os"
)

type registerKey struct {
	owner string
	key   string
}

func keyIsSlab(key string) bool {
	return len(key) > 0 && key[0] == '$'
}

// encode key
func (key registerKey) fromMangled() registerKey {
	a := flow.BytesToAddress([]byte(key.owner))
	var keyString string
	// if slab
	if keyIsSlab(key.key) {
		keyString = "$" + hex.EncodeToString([]byte(key.key[1:]))
	} else {
		keyString = key.key
	}

	return registerKey{
		owner: a.Hex(),
		key:   keyString,
	}
}

// decode key
func (key registerKey) toMangled() registerKey {
	a := flow.HexToAddress(key.owner)
	var keyString string
	// if slab
	if len(key.key) > 0 && key.key[0] == '$' {
		decoded, err := hex.DecodeString(key.key[1:])
		if err != nil {
			panic(err)
		}
		keyString = "$" + string(decoded)
	} else {
		keyString = string(key.key)
	}

	return registerKey{
		owner: string(a.Bytes()),
		key:   keyString,
	}
}

type RemoteRegisterFileCache struct {
	remoteGet   RemoteGetRegisterFunc
	blockHeight uint64
	registers   map[registerKey]flow.RegisterValue

	log zerolog.Logger
}

func NewRemoteRegisterFileCache(
	remoteGet RemoteGetRegisterFunc,
	blockHeight uint64,
	log zerolog.Logger,
) (*RemoteRegisterFileCache, error) {
	c := &RemoteRegisterFileCache{
		remoteGet:   remoteGet,
		blockHeight: blockHeight,
		log:         log,
	}
	err := c.open()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RemoteRegisterFileCache) Get(owner, key string) (flow.RegisterValue, error) {
	val, found := c.registers[registerKey{owner, key}]
	if found {
		return val, nil
	}
	val, err := c.remoteGet(owner, key)
	if err != nil {
		return nil, err
	}
	c.registers[registerKey{owner, key}] = val
	return val, nil
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
		encodedKey := key.fromMangled()
		encodedValue := c.encodeRegisterValue(val)
		err := csvwriter.Write([]string{encodedKey.owner, encodedKey.key, encodedValue})
		if err != nil {
			return err
		}
	}
	return nil
}

// open opens the cache by loading registers from a file
func (c *RemoteRegisterFileCache) open() error {

	c.registers = make(map[registerKey]flow.RegisterValue)

	filename := c.getFilename()
	csvFile, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// file does not exist
			c.log.Info().Msgf("cache file does not exist: %s", filename)
			return nil
		}
		return err
	}
	c.log.Info().Msgf("opening cache file: %s", filename)
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
		decodedKey := registerKey{owner, key}.toMangled()
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
