package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

var addArgs struct {
	prefix  net.IPNet
	segs    []net.IP
	reserve int
	dev     string
}

type SEG6EncapWithTLV struct {
	base    *netlink.SEG6Encap
	reserve int
}

func (s *SEG6EncapWithTLV) Type() int { return s.base.Type() }

func (s *SEG6EncapWithTLV) Decode(_ []byte) error {
	return errors.New("not implemented")
}

func (s *SEG6EncapWithTLV) Encode() ([]byte, error) {
	segments := s.base.Segments
	mode := s.base.Mode

	nsegs := len(segments) // nsegs: number of segments
	if nsegs == 0 {
		return nil, errors.New("EncodeSEG6Encap: No Segment in srh")
	}

	if s.reserve%8 != 0 {
		return nil, errors.New("EncodeSEG6Encap: reserve must be a multiple of 8")
	}
	hdrLen := (16*nsegs + s.reserve) >> 3
	if hdrLen > 255 {
		return nil, errors.New("EncodeSEG6Encap: SRH is too large")
	}

	b := make([]byte, 12, 12+len(segments)*16+s.reserve)
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
	for i := len(segments) - 1; i >= 0; i-- {
		netIP := segments[i]
		segment := netIP.To16()
		if segment == nil || netIP.To4() != nil {
			return nil, fmt.Errorf("EncodeSEG6Encap: segment %q is not an IPv6 address", netIP.String())
		}
		b = append(b, segment...) // srh.Segments
	}

	if s.reserve > 0 {
		tlv := make([]byte, s.reserve)
		b = append(b, tlv...) // srh TLV (optional)
	}

	hdr := make([]byte, 4)
	native.PutUint16(hdr, uint16(len(b)+4))
	native.PutUint16(hdr[2:], nl.SEG6_IPTUNNEL_SRH)

	b = append(hdr, b...) // srh header + srh data
	return b, nil
}

func (s *SEG6EncapWithTLV) String() string {
	return s.base.String()
}

func (s *SEG6EncapWithTLV) Equal(e netlink.Encap) bool {
	other, ok := e.(*SEG6EncapWithTLV)
	return ok && s.base.Equal(other.base) && s.reserve == other.reserve
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add route with SRv6 TLV",
	RunE: func(cmd *cobra.Command, args []string) error {
		nl.EnableErrorMessageReporting = true

		link, err := netlink.LinkByName(addArgs.dev)
		if err != nil {
			return fmt.Errorf("get link %q: %w", addArgs.dev, err)
		}
		route := &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       &addArgs.prefix,
			Protocol:  unix.RTPROT_STATIC,
			Encap: &SEG6EncapWithTLV{
				base: &netlink.SEG6Encap{
					Mode:     nl.SEG6_IPTUN_MODE_ENCAP,
					Segments: addArgs.segs,
				},
				reserve: addArgs.reserve,
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
	addCmd.Flags().IntVar(&addArgs.reserve, "reserve", 0, "Reserved bytes for SRv6 TLV")
	addCmd.Flags().StringVar(&addArgs.dev, "dev", "", "Device to add the route on")
	addCmd.MarkFlagRequired("prefix")
	addCmd.MarkFlagRequired("segs")
	addCmd.MarkFlagRequired("dev")
}
