package EDNS0FRAG

import (
	"github.com/miekg/dns"
	"strconv"
)

// XXX: copied from dns/msg.go since it is not exported
func unpackUint16(msg []byte, off int) (uint16, int) {
        return uint16(msg[off])<<8 | uint16(msg[off+1]), off + 2
}

// XXX: copied from dns/msg.go since it is not exported
func packUint16(i uint16) (byte, byte) {
        return byte(i >> 8), byte(i)
}

// custom EDNS0 option
const (
	EDNS0FRAG = dns.EDNS0LOCALSTART
)

type EDNS0_FRAG struct {
	Code      uint16	// Always EDNS0FRAG
	NumFrag   uint8		// Total number of fragments to expect
	ThisFrag  uint8		// Sequence number of this fragment (0 to NumFrag-1)
}

func (e *EDNS0_FRAG) Option() uint16 { return EDNS0FRAG }
func (e *EDNS0_FRAG) String() string {
	return strconv.FormatInt(int64(e.Option()), 10) + " " + strconv.FormatInt(int64(e.NumFrag), 10) + " " + strconv.FormatInt(int64(e.ThisFrag), 10)
}

func (e *EDNS0_FRAG) unpack(b []byte) error {
	if len(b) != 2 {
		return dns.ErrBuf
	}
	e.NumFrag = b[0]
	e.ThisFrag = b[1]
	return nil
}

func (e *EDNS0_FRAG) pack() ([]byte, error) {
	b := make([]byte, 2)
	b[0] = e.NumFrag
	b[1] = e.ThisFrag
	return b, nil
}
