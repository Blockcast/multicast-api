package fec

import (
	"bytes"
	"fmt"
	"sync/atomic"
)

type BlockingStructure struct {
	N             atomic.Uint32 // The total number of source srcBlks into which the object shall be partitioned.
	I             atomic.Uint32 // The first number of larger source srcBlks
	A             atomic.Uint32 // The larger source block length in symbols, this is for I number of source block
	ASmall        atomic.Uint32 // The smaller source block length in symbols, this is for N - I number of source block
	ESLen         uint16
	T             uint64 // Transport object length
	TransferLen   atomic.Uint64
	MaxSbLen      uint32
	MaxNumEs      uint32
	NumEsPerGroup uint32
	//mux           sync.RWMutex
}

func (bs *BlockingStructure) String() string {
	if bs == nil {
		return "nil"
	}
	buf := bytes.NewBuffer([]byte(fmt.Sprintf("len=%d,N=%d,esLen=%d,maxSb=%d", bs.TransferLen.Load(), bs.N.Load(), bs.ESLen, bs.MaxSbLen)))
	//buf := bytes.NewBuffer([]byte(fmt.Sprintf("len=%d,N=%d,I=%d,A+=%d,A-=%d,eslen=%d,esPerGroup=%d", bs.TransferLen.Load(), bs.N.Load(), bs.I.Load(), bs.A.Load(), bs.ASmall.Load(), bs.ESLen, bs.NumEsPerGroup)))
	//enc := xml.NewEncoder(buf)
	//enc.I.Load()ndent(">", " ")
	//if err := enc.Encode(bs); err != nil {
	//	buf.Write([]byte(fmt.Sprintf("XML ENCODER ERROR: %v", err)))
	//}
	return buf.String()
}

func NewBlockingStructure5052(transferObjLen uint64, maxSourceBlockLen uint32, encodingSymbolLen uint16, numEsPerGroup uint32, stream bool) (*BlockingStructure, error) {
	bs := BlockingStructure{}
	err := UpdateBlockingStructure5052(&bs, transferObjLen, maxSourceBlockLen, encodingSymbolLen, numEsPerGroup, stream)
	if err != nil {
		return nil, err
	}
	return &bs, err
}

// This function calculates blocking scheme parameters as defined in RFC5052 9.1
//
//	B  -- Maximum SourceAddr Block Length, i.e., the maximum number of source
//	      symbols per source block
//	L  -- Transfer Length in octets
//	E  -- Encoding Symbol Length in octets
func UpdateBlockingStructure5052(bs *BlockingStructure, transferObjLen uint64, maxSourceBlockLen uint32, encodingSymbolLen uint16, numEsPerGroup uint32, stream bool) error {
	if bs == nil {
		return fmt.Errorf("nil blocking structure")
	}
	if transferObjLen == 0 {
		return fmt.Errorf("length must be greater than zero")
	}
	//bs.mux.Lock()
	//defer bs.mux.Unlock()
	if bs.ESLen != encodingSymbolLen {
		bs.ESLen = encodingSymbolLen
	}
	if bs.MaxSbLen != maxSourceBlockLen {
		bs.MaxSbLen = maxSourceBlockLen
	}
	if bs.NumEsPerGroup != numEsPerGroup {
		bs.NumEsPerGroup = numEsPerGroup
	}
	if bs.MaxNumEs == 0 {
		bs.MaxNumEs = maxSourceBlockLen
	}
	bs.ASmall.Store(maxSourceBlockLen)
	bs.A.Store(maxSourceBlockLen)
	bs.TransferLen.Store(transferObjLen)
	if !stream {
		return bs.recompute()
	} else {
		bs.T = (bs.TransferLen.Load() + uint64(bs.ESLen) - 1) / uint64(bs.ESLen) // the number of source symbols in the object
		bs.N.Store(uint32((bs.T + uint64(bs.MaxSbLen) - 1) / uint64(bs.MaxSbLen)))
		bs.I.Store(bs.N.Load())
	}
	return nil
}

func (bs *BlockingStructure) recompute() error {
	if bs.MaxSbLen == 0 {
		return fmt.Errorf("maxSourceBlockLen must be greater than zero")
	}
	if bs.ESLen == 0 {
		return fmt.Errorf("encodingSymbolLen must be greater than zero")
	}
	if bs.NumEsPerGroup == 0 {
		return fmt.Errorf("numEsPerGroup must be greater than zero")
	}
	if bs.NumEsPerGroup > bs.MaxSbLen {
		return fmt.Errorf("numEsPerGroup %d must be less than or equal to maxSourceBlockLen %d", bs.NumEsPerGroup, bs.MaxSbLen)
	}
	bs.T = (bs.TransferLen.Load() + uint64(bs.ESLen) - 1) / uint64(bs.ESLen) // the number of source symbols in the object
	bs.N.Store(uint32((bs.T + uint64(bs.MaxSbLen) - 1) / uint64(bs.MaxSbLen)))

	if bs.TransferLen.Load() == 0 {
		return fmt.Errorf("transferLen must be greater than zero")
	}
	if bs.ESLen > 1 {
		A := bs.T / uint64(bs.N.Load())
		bs.A.Store(uint32(A))
		bs.ASmall.Store(bs.A.Load())
		bs.I.Store(uint32(bs.T % uint64(bs.N.Load())))
		if bs.I.Load() > 0 {
			bs.A.Add(1)
		}
	} else {
		bs.A.Store(bs.MaxSbLen)
		bs.ASmall.Store(bs.MaxSbLen)
	}

	return nil
}

func (bs *BlockingStructure) NumSrcSym(sbn uint32) uint32 {
	if sbn >= bs.N.Load() {
		return 0
	}
	num := bs.ASmall.Load()
	if sbn < bs.I.Load() {
		num = bs.A.Load()
	}
	if sbn == bs.N.Load()-1 {
		offset := bs.SrcOffset(sbn)
		end := offset + uint64(bs.ESLen)*uint64(num)
		if end > bs.TransferLen.Load() {
			num = (uint32(bs.TransferLen.Load()-offset) + uint32(bs.ESLen) - 1) / uint32(bs.ESLen)
		}
	}
	return num
}

//	func (bs *BlockingStructure) HasRpr() bool {
//		return  (uint64(bs.MaxNumEs) - uint64(bs.MaxSbLen)) > 0
//	}
func (bs *BlockingStructure) RprBlockSize() uint64 {
	return (uint64(bs.MaxNumEs) - uint64(bs.MaxSbLen)) * uint64(bs.ESLen)
	//return (uint64(bs.MaxNumEs) - uint64(bs.ASmall.Load())) * uint64(bs.ESLen)
}

func (bs *BlockingStructure) RprOffset(sbn uint32) uint64 {
	rprBlockLen := bs.RprBlockSize()
	if sbn > bs.N.Load() {
		sbn = bs.N.Load()
	}
	return rprBlockLen * uint64(sbn)
}

// GetBlockOffset computes the source block offset from the start of the file
func (bs *BlockingStructure) SrcOffset(sbn uint32) uint64 {
	if sbn >= bs.N.Load() {
		return uint64(bs.TransferLen.Load())
	} else if sbn < bs.I.Load() {
		/*if !finalPos {
			return uuint64(cto.ESLen) * uuint64(cto.MaxNumEs) * uuint64(sbn)
		} else {
			return uuint64(cto.ESLen) * uuint64(bs.A.Load()) * uuint64(sbn)
		}*/
		return uint64(bs.ESLen) * uint64(bs.A.Load()) * uint64(sbn)
	} else {
		return uint64(bs.ESLen) * (uint64(bs.I.Load())*uint64(bs.A.Load()) +
			uint64(sbn-bs.I.Load())*uint64(bs.ASmall.Load()))
	}
}

func (bs *BlockingStructure) SourceSBN(offset uint64) uint32 {
	sbn := offset / uint64(bs.ESLen)
	largeBlocksize := uint64(bs.A.Load()) * uint64(bs.I.Load())
	if sbn < largeBlocksize {
		sbn /= uint64(bs.A.Load())
	} else {
		sbn -= largeBlocksize
		sbn /= uint64(bs.ASmall.Load())
		sbn += uint64(bs.I.Load())
	}
	if sbn < uint64(bs.N.Load()) {
		return uint32(sbn)
	}
	return bs.N.Load() - 1
}

func (bs *BlockingStructure) RepairSBN(offset uint64) uint32 {
	rprBlockLen := bs.RprBlockSize()
	return uint32(offset / rprBlockLen)
}

func (bs *BlockingStructure) SrcBlockSize(sbn uint32) (blockLen uint64) {
	if sbn == bs.N.Load()-1 {
		offset := bs.SrcOffset(sbn)
		return bs.TransferLen.Load() - offset
	} else if sbn < bs.I.Load() {
		return uint64(bs.ESLen) * uint64(bs.A.Load())
	} else if sbn < bs.N.Load()-1 {
		return uint64(bs.ESLen) * uint64(bs.ASmall.Load())
	}
	return 0
}

func (bs *BlockingStructure) UpdateLength(length uint64, stream bool) error {
	return UpdateBlockingStructure5052(bs, length, bs.MaxSbLen, bs.ESLen, bs.NumEsPerGroup, stream)
}
