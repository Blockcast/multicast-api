package models

import (
	"encoding/xml"
	"strings"
)

const (
	Pull ContentAcquisitionMethodType = "pull"
	Push                              = "push"

	None                  TransportSecurityType = "none"
	Integrity                                   = "integrity"
	IntegrityAuthenticity                       = "integrityAndAuthenticity"
)

func (t *ContentAcquisitionMethodType) UnmarshalText(text []byte) error {
	switch ContentAcquisitionMethodType(text) {
	case Push:
		*t = Push
	default:
		*t = Pull
	}
	return nil
}

type ServiceComponentIdentifierType struct {
	*DASHComponentIdentifierType
	*HLSComponentIdentifierType
}

func (s *ServiceComponentIdentifierType) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Attributes
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			var overlay any
			switch strings.ToLower(attr.Value) {
			case "dashcomponentidentifiertype":
				s.DASHComponentIdentifierType = &DASHComponentIdentifierType{}
				overlay = s.DASHComponentIdentifierType
			case "hlscomponentidentifiertype":
				s.HLSComponentIdentifierType = &HLSComponentIdentifierType{}
				overlay = s.HLSComponentIdentifierType
			default:

			}
			if err := d.DecodeElement(&overlay, &start); err != nil {
				return err
			}
			break
		}
	}
	return nil
}
