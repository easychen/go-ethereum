// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package storage

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// NetStore implements the ChunkStore interface,
// this chunk access layer assumed 2 chunk stores
// local storage eg. LocalStore and network storage eg., NetStore
// access by calling network is blocking with a timeout
type NetStore struct {
	localStore *LocalStore
	retrieve   func(chunk *Chunk) error
}

func NewNetStore(localStore *LocalStore, retrieve func(chunk *Chunk) error) *NetStore {
	return &NetStore{localStore, retrieve}
}

// Get is the entrypoint for local retrieve requests
// waits for response or times out
func (self *NetStore) Get(key Key) (chunk *Chunk, err error) {
	var created bool
	chunk, created = self.localStore.GetOrCreateRequest(key)
	if chunk.ReqC == nil {
		log.Trace(fmt.Sprintf("DPA.Get: %v found locally, %d bytes", key.Log(), len(chunk.SData)))
		return
	}

	if created {
		if err := self.retrieve(chunk); err != nil {
			return nil, err
		}
	}
	t := time.NewTicker(searchTimeout)
	defer t.Stop()

	select {
	case <-t.C:
		log.Trace(fmt.Sprintf("DPA.Get: %v request time out ", key.Log()))
		return nil, notFound
	case <-chunk.ReqC:
	}
	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))
	return chunk, nil
}

// Put is the entrypoint for local store requests coming from storeLoop
func (self *NetStore) Put(chunk *Chunk) {
	self.localStore.Put(chunk)
}

// Close chunk store
func (self *NetStore) Close() {}
