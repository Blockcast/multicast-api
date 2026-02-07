package api

// FEC type definitions extracted for TinyGo compatibility.
// These types have no OS-specific or TinyGo-incompatible dependencies.

type FECInstance uint16
type FECEncoding uint8
type CodePoint uint8

const (
	ReedSolomonFECInst FECInstance = 0 // Reed-Solomon instance id, when Small Block Systematic FEC scheme is used

	// Fully specified
	COM_NO_C_FEC_ENC_ID FECEncoding = 0   // Compact No-Code FEC scheme
	RS_GEN_FEC_ENC_ID   FECEncoding = 2   // Reed-Solomon FEC scheme RFC5510, over GF(2^^m) where m=8 or 16
	RS_GF8_FEC_ENC_ID   FECEncoding = 5   // Reed-Solomon FEC scheme RFC5510, over GF(2^^8)
	RAPTORQ_FEC_ENC_ID  FECEncoding = 6   // RaptorQ FEC scheme RFC6330
	SB_LB_E_FEC_ENC_ID  FECEncoding = 128 // Small Block, Large Block and Expandable FEC scheme
	SB_SYS_FEC_ENC_ID   FECEncoding = 129 // Small Block Systematic FEC scheme
	COM_FEC_ENC_ID       FECEncoding = 130 // Compact FEC scheme
)

func (d FECEncoding) NamedEnum() ([]interface{}, []string) {
	return []interface{}{
			COM_NO_C_FEC_ENC_ID,
			RS_GF8_FEC_ENC_ID,
			RAPTORQ_FEC_ENC_ID},
		[]string{
			"Compact-No-Code",
			"Reed-Solomon-GF(2^^8)",
			"RaptorQ",
		}
}

func (s FECEncoding) String() string {
	switch s {
	case COM_NO_C_FEC_ENC_ID:
		return "Compact-No-Code"
	case RS_GF8_FEC_ENC_ID:
		return "Reed-Solomon-GF(2^^8)"
	case RAPTORQ_FEC_ENC_ID:
		return "RaptorQ"
	default:
		return "unknown"
	}
}
