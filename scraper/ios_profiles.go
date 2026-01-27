package scraper

import (
	"github.com/bogdanfinn/fhttp/http2"
	tls "github.com/bogdanfinn/utls"
	"github.com/bogdanfinn/tls-client/profiles"
)

// iOS Profile ClientHelloIDs

var HelloStandardIOS = tls.ClientHelloID{
	Client:               "StandardIOS",
	RandomExtensionOrder: false,
	Version:              "1.0.0",
	Seed:                 nil,
	SpecFactory:          standardIOSSpec,
}

var HelloSecondaryIOS = tls.ClientHelloID{
	Client:               "SecondaryIOS",
	RandomExtensionOrder: false,
	Version:              "1.0.0",
	Seed:                 nil,
	SpecFactory:          secondaryIOSSpec,
}

var HelloSecondaryIOS26 = tls.ClientHelloID{
	Client:               "SecondaryIOS26",
	RandomExtensionOrder: false,
	Version:              "1.0.0",
	Seed:                 nil,
	SpecFactory:          secondaryIOS26Spec,
}

var HelloStandardIOS18 = tls.ClientHelloID{
	Client:               "StandardIOS18",
	RandomExtensionOrder: false,
	Version:              "1.0.0",
	Seed:                 nil,
	SpecFactory:          standardIOS18Spec,
}

// iOS Client Profiles with HTTP/2 settings

var StandardIOS = profiles.NewClientProfile(
	HelloStandardIOS,
	map[http2.SettingID]uint32{
		http2.SettingEnablePush:           0,
		http2.SettingInitialWindowSize:    2097152,
		http2.SettingMaxConcurrentStreams: 100,
	},
	[]http2.SettingID{
		http2.SettingEnablePush,
		http2.SettingInitialWindowSize,
		http2.SettingMaxConcurrentStreams,
	},
	[]string{":method", ":scheme", ":path", ":authority"},
	10485760,
	[]http2.Priority{},
	&http2.PriorityParam{},
)

var SecondaryIOS = profiles.NewClientProfile(
	HelloSecondaryIOS,
	map[http2.SettingID]uint32{
		http2.SettingEnablePush:           0,
		http2.SettingInitialWindowSize:    2097152,
		http2.SettingMaxConcurrentStreams: 100,
	},
	[]http2.SettingID{
		http2.SettingEnablePush,
		http2.SettingInitialWindowSize,
		http2.SettingMaxConcurrentStreams,
	},
	[]string{":method", ":scheme", ":path", ":authority"},
	10485760,
	[]http2.Priority{},
	&http2.PriorityParam{},
)

var SecondaryIOS26 = profiles.NewClientProfile(
	HelloSecondaryIOS26,
	map[http2.SettingID]uint32{
		http2.SettingEnablePush:           0,
		http2.SettingInitialWindowSize:    2097152,
		http2.SettingMaxConcurrentStreams: 100,
		9:                                 1, // Unknown setting ID 9
	},
	[]http2.SettingID{
		http2.SettingEnablePush,
		http2.SettingInitialWindowSize,
		http2.SettingMaxConcurrentStreams,
		9,
	},
	[]string{":method", ":scheme", ":path", ":authority"},
	10485760,
	[]http2.Priority{},
	&http2.PriorityParam{},
)

var StandardIOS18 = profiles.NewClientProfile(
	HelloStandardIOS18,
	map[http2.SettingID]uint32{
		http2.SettingEnablePush:           0,
		http2.SettingInitialWindowSize:    2097152,
		http2.SettingMaxConcurrentStreams: 100,
	},
	[]http2.SettingID{
		http2.SettingEnablePush,
		http2.SettingInitialWindowSize,
		http2.SettingMaxConcurrentStreams,
	},
	[]string{":method", ":scheme", ":path", ":authority"},
	10485760,
	[]http2.Priority{},
	&http2.PriorityParam{},
)

// TLS Spec Functions

func standardIOSSpec() (tls.ClientHelloSpec, error) {
	return tls.ClientHelloSpec{
		CipherSuites: []uint16{
			tls.GREASE_PLACEHOLDER,
			0x1301, // TLS_AES_128_GCM_SHA256
			0x1302, // TLS_AES_256_GCM_SHA384
			0x1303, // TLS_CHACHA20_POLY1305_SHA256
			0xc02c, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
			0xc02b, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
			0xcca9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
			0xc030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
			0xc02f, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
			0xcca8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
			0xc00a, // TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
			0xc009, // TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
			0xc014, // TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
			0xc013, // TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
		},
		CompressionMethods: []uint8{tls.CompressionNone},
		Extensions: []tls.TLSExtension{
			&tls.UtlsGREASEExtension{},
			&tls.SNIExtension{},
			&tls.ExtendedMasterSecretExtension{},
			&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateNever},
			&tls.SupportedCurvesExtension{Curves: []tls.CurveID{
				tls.CurveID(tls.GREASE_PLACEHOLDER),
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
				tls.CurveP521,
			}},
			&tls.SupportedPointsExtension{SupportedPoints: []byte{0}},
			&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
			&tls.StatusRequestExtension{},
			&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
				tls.ECDSAWithP256AndSHA256,
				tls.PSSWithSHA256,
				tls.PKCS1WithSHA256,
				tls.ECDSAWithP384AndSHA384,
				tls.PSSWithSHA384,
				tls.PSSWithSHA384,
				tls.PKCS1WithSHA384,
				tls.PSSWithSHA512,
				tls.PKCS1WithSHA512,
				tls.PKCS1WithSHA1,
			}},
			&tls.SCTExtension{},
			&tls.KeyShareExtension{KeyShares: []tls.KeyShare{
				{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
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
				tls.CertCompressionZlib,
			}},
			&tls.UtlsGREASEExtension{},
			&tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
		},
	}, nil
}

func secondaryIOSSpec() (tls.ClientHelloSpec, error) {
	return tls.ClientHelloSpec{
		CipherSuites: []uint16{
			tls.GREASE_PLACEHOLDER,
			0x1301, // TLS_AES_128_GCM_SHA256
			0x1302, // TLS_AES_256_GCM_SHA384
			0x1303, // TLS_CHACHA20_POLY1305_SHA256
			0xc02c, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
			0xc02b, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
			0xcca9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
			0xc030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
			0xc02f, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
			0xcca8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
			0xc00a, // TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
			0xc009, // TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
			0xc014, // TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
			0xc013, // TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
			0x9d,   // TLS_RSA_WITH_AES_256_GCM_SHA384
			0x9c,   // TLS_RSA_WITH_AES_128_GCM_SHA256
			0x35,   // TLS_RSA_WITH_AES_256_CBC_SHA
			0x2f,   // TLS_RSA_WITH_AES_128_CBC_SHA
			0xc008, // TLS_ECDHE_ECDSA_WITH_3DES_EDE_CBC_SHA
			0xc012, // TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA
			0xa,    // TLS_RSA_WITH_3DES_EDE_CBC_SHA
		},
		CompressionMethods: []uint8{tls.CompressionNone},
		Extensions: []tls.TLSExtension{
			&tls.UtlsGREASEExtension{},
			&tls.SNIExtension{},
			&tls.ExtendedMasterSecretExtension{},
			&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateOnceAsClient},
			&tls.SupportedCurvesExtension{Curves: []tls.CurveID{
				tls.CurveID(tls.GREASE_PLACEHOLDER),
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
				tls.CurveP521,
			}},
			&tls.SupportedPointsExtension{SupportedPoints: []byte{
				tls.PointFormatUncompressed,
			}},
			&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
			&tls.StatusRequestExtension{},
			&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
				tls.ECDSAWithP256AndSHA256,
				tls.PSSWithSHA256,
				tls.PKCS1WithSHA256,
				tls.ECDSAWithP384AndSHA384,
				tls.PSSWithSHA384,
				tls.PSSWithSHA384,
				tls.PKCS1WithSHA384,
				tls.PSSWithSHA512,
				tls.PKCS1WithSHA512,
				tls.PKCS1WithSHA1,
			}},
			&tls.SCTExtension{},
			&tls.KeyShareExtension{KeyShares: []tls.KeyShare{
				{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
				{Group: tls.X25519},
			}},
			&tls.PSKKeyExchangeModesExtension{Modes: []uint8{
				tls.PskModeDHE,
			}},
			&tls.SupportedVersionsExtension{Versions: []uint16{
				tls.GREASE_PLACEHOLDER,
				tls.VersionTLS13,
				tls.VersionTLS12,
				tls.VersionTLS11,
				tls.VersionTLS10,
			}},
			&tls.UtlsCompressCertExtension{Algorithms: []tls.CertCompressionAlgo{
				tls.CertCompressionZlib,
			}},
			&tls.UtlsGREASEExtension{},
			&tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
		},
	}, nil
}

func secondaryIOS26Spec() (tls.ClientHelloSpec, error) {
	return tls.ClientHelloSpec{
		CipherSuites: []uint16{
			tls.GREASE_PLACEHOLDER,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			0xc008, // TLS_ECDHE_ECDSA_WITH_3DES_EDE_CBC_SHA
			0xc012, // TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA
			0xa,    // TLS_RSA_WITH_3DES_EDE_CBC_SHA
		},
		CompressionMethods: []uint8{
			tls.CompressionNone,
		},
		Extensions: []tls.TLSExtension{
			&tls.UtlsGREASEExtension{},
			&tls.SNIExtension{},
			&tls.ExtendedMasterSecretExtension{},
			&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateNever},
			&tls.SupportedCurvesExtension{Curves: []tls.CurveID{
				tls.CurveID(tls.GREASE_PLACEHOLDER),
				tls.X25519MLKEM768,
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
				tls.CurveP521,
			}},
			&tls.SupportedPointsExtension{SupportedPoints: []byte{0x00}},
			&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
			&tls.StatusRequestExtension{},
			&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
				tls.ECDSAWithP256AndSHA256,
				tls.PSSWithSHA256,
				tls.PKCS1WithSHA256,
				tls.ECDSAWithP384AndSHA384,
				tls.PSSWithSHA384,
				tls.PSSWithSHA384,
				tls.PKCS1WithSHA384,
				tls.PSSWithSHA512,
				tls.PKCS1WithSHA512,
				tls.PKCS1WithSHA1,
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
				tls.CertCompressionZlib,
			}},
			&tls.UtlsGREASEExtension{},
		},
	}, nil
}

func standardIOS18Spec() (tls.ClientHelloSpec, error) {
	return tls.ClientHelloSpec{
		CipherSuites: []uint16{
			tls.GREASE_PLACEHOLDER,
			0x1301, // TLS_AES_128_GCM_SHA256
			0x1302, // TLS_AES_256_GCM_SHA384
			0x1303, // TLS_CHACHA20_POLY1305_SHA256
			0xc02c, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
			0xc02b, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
			0xcca9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
			0xc030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
			0xc02f, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
			0xcca8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
			0xc00a, // TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
			0xc009, // TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
			0xc014, // TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
			0xc013, // TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
		},
		CompressionMethods: []uint8{tls.CompressionNone},
		Extensions: []tls.TLSExtension{
			&tls.UtlsGREASEExtension{},
			&tls.SNIExtension{},
			&tls.ExtendedMasterSecretExtension{},
			&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateNever},
			&tls.SupportedCurvesExtension{Curves: []tls.CurveID{
				tls.CurveID(tls.GREASE_PLACEHOLDER),
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
				tls.CurveP521,
			}},
			&tls.SupportedPointsExtension{SupportedPoints: []byte{0}},
			&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
			&tls.StatusRequestExtension{},
			&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
				tls.ECDSAWithP256AndSHA256,
				tls.PSSWithSHA256,
				tls.PKCS1WithSHA256,
				tls.ECDSAWithP384AndSHA384,
				tls.PSSWithSHA384,
				tls.PSSWithSHA384,
				tls.PKCS1WithSHA384,
				tls.PSSWithSHA512,
				tls.PKCS1WithSHA512,
				tls.PKCS1WithSHA1,
			}},
			&tls.SCTExtension{},
			&tls.KeyShareExtension{KeyShares: []tls.KeyShare{
				{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
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
				tls.CertCompressionZlib,
			}},
			&tls.UtlsGREASEExtension{},
			&tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
		},
	}, nil
}
