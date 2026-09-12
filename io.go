// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cedar

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

// cedarData mirrors Cedar with exported fields; it is the wire format shared
// by the gob and JSON encoders.
type cedarData struct {
	Reduced bool
	Array   []int32 // baseV, check pairs
	NInfos  []byte  // sibling, child pairs
	Blocks  []int32 // prev, next, num, reject, trial, eHead per block
	Reject  [257]int32

	HeadFull, HeadClosed, HeadOpen int32
	Capacity, Size                 int
	Ordered                        bool
	MaxTrial                       int32
}

func (cd *Cedar) toData() *cedarData {
	d := &cedarData{
		Reduced:    cd.Reduced,
		Array:      make([]int32, 0, 2*cd.size),
		NInfos:     make([]byte, 0, 2*cd.size),
		Blocks:     make([]int32, 0, 6*(cd.size>>8)),
		Reject:     cd.reject,
		HeadFull:   cd.blocksHeadFull,
		HeadClosed: cd.blocksHeadClosed,
		HeadOpen:   cd.blocksHeadOpen,
		Capacity:   cd.capacity,
		Size:       cd.size,
		Ordered:    cd.ordered,
		MaxTrial:   cd.maxTrial,
	}
	for _, n := range cd.array[:cd.size] {
		d.Array = append(d.Array, n.baseV, n.check)
	}
	for _, n := range cd.nInfos[:cd.size] {
		d.NInfos = append(d.NInfos, n.sibling, n.child)
	}
	for _, b := range cd.blocks[:cd.size>>8] {
		d.Blocks = append(d.Blocks, b.prev, b.next, b.num, b.reject, b.trial, b.eHead)
	}

	return d
}

func (cd *Cedar) fromData(d *cedarData) error {
	if d.Size < 256 || d.Size%256 != 0 || d.Capacity < d.Size ||
		len(d.Array) != 2*d.Size || len(d.NInfos) != 2*d.Size || len(d.Blocks) != 6*(d.Size>>8) {
		return ErrInvalidData
	}

	*cd = Cedar{
		Reduced:          d.Reduced,
		array:            make([]Node, d.Capacity),
		nInfos:           make([]NInfo, d.Capacity),
		blocks:           make([]Block, d.Capacity>>8),
		reject:           d.Reject,
		blocksHeadFull:   d.HeadFull,
		blocksHeadClosed: d.HeadClosed,
		blocksHeadOpen:   d.HeadOpen,
		capacity:         d.Capacity,
		size:             d.Size,
		ordered:          d.Ordered,
		maxTrial:         d.MaxTrial,
	}
	for i := range cd.array[:d.Size] {
		cd.array[i] = Node{baseV: d.Array[2*i], check: d.Array[2*i+1]}
	}
	for i := range cd.nInfos[:d.Size] {
		cd.nInfos[i] = NInfo{sibling: d.NInfos[2*i], child: d.NInfos[2*i+1]}
	}
	for i := range cd.blocks[:d.Size>>8] {
		b := d.Blocks[6*i : 6*i+6]
		cd.blocks[i] = Block{prev: b[0], next: b[1], num: b[2], reject: b[3], trial: b[4], eHead: b[5]}
	}

	return nil
}

// GobEncode implements gob.GobEncoder.
func (cd *Cedar) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(cd.toData())
	return buf.Bytes(), err
}

// GobDecode implements gob.GobDecoder.
func (cd *Cedar) GobDecode(data []byte) error {
	var d cedarData
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&d); err != nil {
		return err
	}

	return cd.fromData(&d)
}

// MarshalJSON implements json.Marshaler.
func (cd *Cedar) MarshalJSON() ([]byte, error) {
	return json.Marshal(cd.toData())
}

// UnmarshalJSON implements json.Unmarshaler.
func (cd *Cedar) UnmarshalJSON(data []byte) error {
	var d cedarData
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}

	return cd.fromData(&d)
}
