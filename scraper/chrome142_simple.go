package scraper

import (
	"github.com/bogdanfinn/fhttp/http2"
	tls "github.com/bogdanfinn/utls"
	"github.com/bogdanfinn/tls-client/profiles"
)

// HelloChrome_142 - Custom ClientHelloID for Chrome 142
// Uses utlsIdToSpec from u_parrots.go which is identical to Chrome_131/133
var HelloChrome_142 = tls.ClientHelloID{
	Client:      "Chrome",
	Version:     "142",
	Seed:        nil,
	SpecFactory: nil, // Will use utlsIdToSpec from library
}

// Chrome142Simple - Chrome 142 profile based on real fingerprint
var Chrome142Simple = profiles.NewClientProfile(
	HelloChrome_142,
	// HTTP/2 SETTINGS from fingerprint: 1:65536;2:0;4:6291456;6:262144
	map[http2.SettingID]uint32{
		http2.SettingHeaderTableSize:      65536,
		http2.SettingEnablePush:           0,
		http2.SettingInitialWindowSize:    6291456,
		http2.SettingMaxHeaderListSize:    262144,
	},
	[]http2.SettingID{
		http2.SettingHeaderTableSize,
		http2.SettingEnablePush,
		http2.SettingInitialWindowSize,
		http2.SettingMaxHeaderListSize,
	},
	// Pseudo header order: m,a,s,p
	[]string{":method", ":authority", ":scheme", ":path"},
	// Connection flow: 15663105
	15663105,
	[]http2.Priority{
		{StreamID: 3, PriorityParam: http2.PriorityParam{StreamDep: 0, Exclusive: false, Weight: 200}},
		{StreamID: 5, PriorityParam: http2.PriorityParam{StreamDep: 0, Exclusive: false, Weight: 100}},
		{StreamID: 7, PriorityParam: http2.PriorityParam{StreamDep: 0, Exclusive: false, Weight: 0}},
		{StreamID: 9, PriorityParam: http2.PriorityParam{StreamDep: 7, Exclusive: false, Weight: 0}},
		{StreamID: 11, PriorityParam: http2.PriorityParam{StreamDep: 3, Exclusive: false, Weight: 0}},
		{StreamID: 13, PriorityParam: http2.PriorityParam{StreamDep: 0, Exclusive: false, Weight: 240}},
	},
	&http2.PriorityParam{StreamDep: 13, Exclusive: false, Weight: 41},
)

// chrome142Spec - EXACT clone of Chrome_133 TLS spec
func chrome142Spec() (tls.ClientHelloSpec, error) {
	return tls.ClientHelloSpec{
		CipherSuites: []uint16{
			tls.GREASE_PLACEHOLDER,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		CompressionMethods: []byte{0x00},
		// EXACT extension order from Chrome_133 with ShuffleChromeTLSExtensions
		Extensions: tls.ShuffleChromeTLSExtensions([]tls.TLSExtension{
			&tls.UtlsGREASEExtension{},
			&tls.SNIExtension{},
			&tls.ExtendedMasterSecretExtension{},
			&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateOnceAsClient},
			&tls.SupportedCurvesExtension{Curves: []tls.CurveID{
				tls.GREASE_PLACEHOLDER,
				tls.X25519MLKEM768,
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
			}},
			&tls.SupportedPointsExtension{SupportedPoints: []byte{0x00}},
			&tls.SessionTicketExtension{},
			&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
			&tls.StatusRequestExtension{},
			&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
				tls.ECDSAWithP256AndSHA256,
				tls.PSSWithSHA256,
				tls.PKCS1WithSHA256,
				tls.ECDSAWithP384AndSHA384,
				tls.PSSWithSHA384,
				tls.PKCS1WithSHA384,
				tls.PSSWithSHA512,
				tls.PKCS1WithSHA512,
			}},
			&tls.SCTExtension{},
			&tls.KeyShareExtension{KeyShares: []tls.KeyShare{
				{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
				{Group: tls.X25519MLKEM768},
				{Group: tls.X25519},
			}},
			&tls.PSKKeyExchangeModesExtension{Modes: []uint8{
				tls.PskModeDHE,
			}},
			&tls.SupportedVersionsExtension{Versions: []uint16{
				tls.GREASE_PLACEHOLDER,
				tls.VersionTLS13,
				tls.VersionTLS12,
			}},
			&tls.UtlsCompressCertExtension{Algorithms: []tls.CertCompressionAlgo{
				tls.CertCompressionBrotli,
			}},
			&tls.ApplicationSettingsExtension{SupportedProtocols: []string{"h2"}},
			tls.BoringGREASEECH(),
			&tls.UtlsGREASEExtension{},
		}),
	}, nil
}
