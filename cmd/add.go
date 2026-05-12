package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"

	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

var addArgs struct {
	prefix net.IPNet
	segs   []net.IP
	dev    string

	tlvType int
	tlvLen  int
}

type tlvConfig struct {
	typ int
	len int
}

type seg6EncapWithTLV struct {
	base *netlink.SEG6Encap
	tlv  *tlvConfig
}

func (s *seg6EncapWithTLV) Type() int { return s.base.Type() }

func (s *seg6EncapWithTLV) Decode(_ []byte) error {
	return errors.New("not implemented")
}

func (s *seg6EncapWithTLV) Encode() ([]byte, error) {
	var err error
	segments := s.base.Segments
	mode := s.base.Mode
	nsegs := len(segments) // nsegs: number of segments

	var tlv []byte
	if s.tlv != nil {
		tlv, err = encodeTLV(s.tlv)
		if err != nil {
			return nil, err
		}
	}

	hdrLen := (16*nsegs + len(tlv)) >> 3
	if hdrLen > 255 {
		return nil, errors.New("SRH is too large")
	}

	b := make([]byte, 12, 12+len(segments)*16+len(tlv))
	native := nl.NativeEndian()
	native.PutUint32(b, uint32(mode))
	b[4] = 0                    // srh.nextHdr (0 when calling netlink)
	b[5] = uint8(hdrLen)        // srh.hdrLen excludes the first 8 octets
	b[6] = nl.IPV6_SRCRT_TYPE_4 // srh.routingType (assigned by IANA)
	b[7] = uint8(nsegs - 1)     // srh.segmentsLeft
	b[8] = uint8(nsegs - 1)     // srh.firstSegment
	b[9] = 0                    // srh.flags (SR6_FLAG1_HMAC for srh_hmac)
	// srh.reserved: Defined as "Tag" in draft-ietf-6man-segment-routing-header-07
	native.PutUint16(b[10:], 0) // srh.reserved
	for _, seg := range slices.Backward(segments) {
		b = append(b, seg...) // srh.Segments
	}

	if len(tlv) > 0 {
		b = append(b, tlv...) // srh TLV (optional)
	}

	hdr := make([]byte, 4)
	native.PutUint16(hdr, uint16(len(b)+4))
	native.PutUint16(hdr[2:], nl.SEG6_IPTUNNEL_SRH)

	b = append(hdr, b...) // srh header + srh data
	return b, nil
}

func (s *seg6EncapWithTLV) String() string {
	return s.base.String()
}

func (s *seg6EncapWithTLV) Equal(e netlink.Encap) bool {
	other, ok := e.(*seg6EncapWithTLV)
	return ok && s.base.Equal(other.base) && s.tlv.typ == other.tlv.typ && s.tlv.len == other.tlv.len
}

func encodeTLV(tlv *tlvConfig) ([]byte, error) {
	// | Type (1 byte) | Length (1 byte) | Value (Length bytes) | Padding to 8-byte boundary |

	used := 2 + tlv.len
	total := roundUp8(used)
	out := make([]byte, total)
	out[0] = byte(tlv.typ)
	out[1] = byte(tlv.len)

	padLen := total - used
	switch {
	case padLen == 1:
		out[used] = 0 // Pad1
	case padLen >= 2:
		out[used] = 4 // PadN
		out[used+1] = byte(padLen - 2)
	}

	return out, nil
}

func roundUp8(n int) int {
	return (n + 7) & ^7
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add route with SRv6 TLV",
	RunE: func(cmd *cobra.Command, args []string) error {
		nl.EnableErrorMessageReporting = true

		var tlv *tlvConfig
		if addArgs.tlvType > 0 && addArgs.tlvLen > 0 {
			tlv = &tlvConfig{
				typ: addArgs.tlvType,
				len: addArgs.tlvLen,
			}
		}

		link, err := netlink.LinkByName(addArgs.dev)
		if err != nil {
			return fmt.Errorf("get link %q: %w", addArgs.dev, err)
		}
		route := &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       &addArgs.prefix,
			Protocol:  unix.RTPROT_STATIC,
			Encap: &seg6EncapWithTLV{
				base: &netlink.SEG6Encap{
					Mode:     nl.SEG6_IPTUN_MODE_ENCAP,
					Segments: addArgs.segs,
				},
				tlv: tlv,
			},
		}
		if err := netlink.RouteAdd(route); err != nil {
			if errors.Is(err, unix.EEXIST) {
				slog.Warn("route already exists, replacing it", "route", route)
				if err = netlink.RouteReplace(route); err != nil {
					return fmt.Errorf("replace route: %w", err)
				}
				return nil
			}
			return fmt.Errorf("add route: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().IPNetVar(&addArgs.prefix, "prefix", net.IPNet{}, "Prefix to add")
	addCmd.Flags().IPSliceVar(&addArgs.segs, "segs", []net.IP{}, "Segments for SRv6")
	addCmd.Flags().IntVar(&addArgs.tlvType, "tlv-type", 0, "SRH TLV type to initialize (1-255)")
	addCmd.Flags().IntVar(&addArgs.tlvLen, "tlv-len", 0, "SRH TLV value length to initialize (0-255)")
	addCmd.Flags().StringVar(&addArgs.dev, "dev", "", "Device to add the route on")
	addCmd.MarkFlagRequired("prefix")
	addCmd.MarkFlagRequired("segs")
	addCmd.MarkFlagRequired("dev")
}
