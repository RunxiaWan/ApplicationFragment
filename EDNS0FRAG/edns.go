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

//add by RunxiaWan application fragmentation's
type EDNS0_Af struct {
	Code    uint16
	SeqNO   uint8
	TotalNo uint8
}

func (e *EDNS0_Af) Option() uint16 { return e.Code }
func (e *EDNS0_Af) String() string {
	return strconv.FormatInt(int64(e.Code), 10) + " SeqNo:" + strconv.FormatInt(int64(e.SeqNO), 10) + " TotalNo:" + strconv.FormatInt(int64(e.TotalNo), 10)
}

func (e *EDNS0_Af) unpack(b []byte) error {
	if len(b) != 32 {
		return dns.ErrBuf
	}
	e.Code, _ = unpackUint16(b, 0)
	e.SeqNO = b[4]
	e.TotalNo = b[5]
	return nil
}

func (e *EDNS0_Af) pack() ([]byte, error) {
	b := make([]byte, 6)
	b[0], b[1] = packUint16(e.Code)
	b[2], b[3] = packUint16(uint16(2))
	b[4] = e.SeqNO
	b[5] = e.TotalNo
	return b, nil
}
