package main

import (
	"fmt"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
)

type registerReadEntry struct {
	key  registerKey
	read int
}

func (e registerReadEntry) String() string {
	return fmt.Sprintf("%v %v: %v bytes", e.key.owner, e.key.key, e.read)
}

type RemoteRegisterReadTracker struct {
	remoteGet RemoteGetRegisterFunc

	registerRead []registerReadEntry

	log zerolog.Logger
}

func NewRemoteRegisterReadTracker(
	remoteGet RemoteGetRegisterFunc,
	log zerolog.Logger,
) *RemoteRegisterReadTracker {
	return &RemoteRegisterReadTracker{
		remoteGet:    remoteGet,
		registerRead: []registerReadEntry{},
		log:          log,
	}
}

func (r *RemoteRegisterReadTracker) Get(owner, key string) (flow.RegisterValue, error) {
	val, err := r.remoteGet(owner, key)
	if err != nil {
		return nil, err
	}
	if keyIsSlab(key) {
		r.registerRead[len(r.registerRead)-1].read += len(val)
	} else {
		r.registerRead = append(r.registerRead, registerReadEntry{
			key:  registerKey{owner, key}.fromMangled(),
			read: len(val),
		})
	}

	return val, nil
}

func (r *RemoteRegisterReadTracker) LogReads() {
	arr := zerolog.Arr()

	for _, entry := range r.registerRead {
		arr = arr.Str(entry.String())
	}

	r.log.Info().
		Array("arr", arr).
		Msg("registers read")
}
