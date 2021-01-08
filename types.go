// Msg Types
//
// All messages received from the gateway are first decoded to the BaseMsg
// type. The BaseMsg type extracts the following JSON from the message:
//		{
//			"janus": <Type>,
//			"transaction": <ID>,
//			"session_id": <Session>,
//			"sender": <Handle>
//		}
// The Type field is inspected to determine which concrete type
// to decode the message to, while the other fields (ID/Session/Handle) are
// inspected to determine where the message should be delivered. Messages
// with an ID field defined are considered responses to previous requests, and
// will be passed directly to requester. Messages without an ID field are
// considered unsolicited events from the gateway and are expected to have
// both Session and Handle fields defined. They will be passed to the Events
// channel of the related Handle and can be read from there.

package janus

var msgtypes = map[string]func() interface{}{
	"error":       func() interface{} { return &ErrorMsg{} },
	"success":     func() interface{} { return &SuccessMsg{} },
	"detached":    func() interface{} { return &DetachedMsg{} },
	"server_info": func() interface{} { return &InfoMsg{} },
	"ack":         func() interface{} { return &AckMsg{} },
	"event":       func() interface{} { return &EventMsg{} },
	"webrtcup":    func() interface{} { return &WebRTCUpMsg{} },
	"media":       func() interface{} { return &MediaMsg{} },
	"hangup":      func() interface{} { return &HangupMsg{} },
	"slowlink":    func() interface{} { return &SlowLinkMsg{} },
	"timeout":     func() interface{} { return &TimeoutMsg{} },
}

type BaseMsg struct {
	Type    string `json:"janus"`
	ID      string `json:"transaction"`
	Session uint64 `json:"session_id"`
	Handle  uint64 `json:"sender"`
}

type ErrorMsg struct {
	Err ErrorData `json:"error"`
}

type ErrorData struct {
	Code   int
	Reason string
}

func (err *ErrorMsg) Error() string {
	return err.Err.Reason
}

type SuccessMsg struct {
	Data       SuccessData
	PluginData PluginData
	Session    uint64 `json:"session_id"`
	Handle     uint64 `json:"sender"`
}

type SuccessData struct {
	ID uint64
}

type DetachedMsg struct{}

type InfoMsg struct {
	Name          string
	Version       int
	VersionString string `json:"version_string"`
	Author        string
	DataChannels  bool   `json:"data_channels"`
	IPv6          bool   `json:"ipv6"`
	LocalIP       string `json:"local-ip"`
	IceTCP        bool   `json:"ice-tcp"`
	Transports    map[string]PluginInfo
	Plugins       map[string]PluginInfo
}

type PluginInfo struct {
	Name          string
	Author        string
	Description   string
	Version       int
	VersionString string `json:"version_string"`
}

type AckMsg struct{}

type EventMsg struct {
	Plugindata PluginData
	Jsep       map[string]interface{}
	Session    uint64 `json:"session_id"`
	Handle     uint64 `json:"sender"`
}

type PluginData struct {
	Plugin string
	Data   map[string]interface{}
}

type WebRTCUpMsg struct {
	Session uint64 `json:"session_id"`
	Handle  uint64 `json:"sender"`
}

type TimeoutMsg struct {
	Session uint64 `json:"session_id"`
}

type SlowLinkMsg struct {
	Uplink bool
	Lost   int64
}

type MediaMsg struct {
	Type      string
	Receiving bool
}

type HangupMsg struct {
	Reason  string
	Session uint64 `json:"session_id"`
	Handle  uint64 `json:"sender"`
}

const (
	Create           = "create"
	Destroy          = "destroy"
	Exists           = "exists"
	Info             = "Info"
	ListParticipants = "listparticipants"
	RtpForward       = "rtp_forward"
)

type GenericRequest struct {
	Request   *string `json:"request,omitempty"`
	Type      *string `json:"type,omitempty"`
	Room      *int64  `json:"room,omitempty"`
	Id        *int    `json:"id,omitempty"`
	Secret    *string `json:"secret,omitempty"`
	Permanent *bool   `json:"permanent,omitempty"`
}

type CreateRequest struct {
	GenericRequest
	IsPrivate          *bool   `json:"is_private,omitempty"`
	Audio              *bool   `json:"audio,omitempty"`
	AudioPort          *int    `json:"audioport,omitempty"`
	AudioRTCPPort      *int    `json:"audiortcpport,omitempty"`
	AudioPT            *int    `json:"audiopt,omitempty"`
	AudioRTPMap        *string `json:"audiortpmap,omitempty"`
	AudioCodec         *string `json:"audiocodec,omitempty"`
	Bitrate            *int64  `json:"bitrate,omitempty"`
	FirFreq            *int    `json:"fir_freq,omitempty"`
	Publishers         *int    `json:"publishers,omitempty"`
	TransportWideCcExt *bool   `json:"transport_wide_cc_ext,omitempty"`
	Video              *bool   `json:"video,omitempty"`
	VideoPort          *int    `json:"videoport,omitempty"`
	VideoRTCPPort      *int    `json:"videortcpport,omitempty"`
	VideoPT            *int    `json:"videopt,omitempty"`
	VideoRTPMap        *string `json:"videortpmap,omitempty"`
	VideoCodec         *string `json:"videocodec,omitempty"`
	VideoOrientExt     *string `json:"videoorient_ext,omitempty"`
}

type RTPForwardRequest struct {
	GenericRequest
	PublisherID   *int    `json:"publisher_id,omitempty"`
	Host          *string `json:"host,omitempty"`
	HostFamily    *string `json:"host_family,omitempty"`
	VideoPort     *int    `json:"video_port,omitempty"`
	VideoRTCPPort *int    `json:"video_rtcp_port,omitempty"`
	AlwaysOn      *bool   `json:"always_on,omitempty"`
}

type Response struct {
	VideoRoom     *string          `json:"videoroom,omitempty"`
	AudioBridge   *string          `json:"audiobridge,omitempty"`
	TextRoom      *string          `json:"textroom,omitempty"`
	Streaming     string           `json:"streaming,omitempty"`
	Room          *int64           `json:"room,omitempty"`
	Exists        *bool            `json:"exists,omitempty"`
	Participants  *[]Participant   `json:"participants,omitempty"`
	Rooms         *[]Room          `json:"list,omitempty"`
	Info          *Room            `json:"info,omitempty"`
	RTPForwarders *[]RTPForwarders `json:"rtp_forwarders,omitempty"`
}

type Participant struct {
	ID        *int64  `json:"id,omitempty"`
	Display   *string `json:"display,omitempty"`
	Publisher *bool   `json:"publisher,omitempty"`
	Talking   *bool   `json:"talking,omitempty"`
	Setup     *bool   `json:"setup,omitempty"`
	Muted     *bool   `json:"muted,omitempty"`
	Username  *string `json:"username,omitempty"`
}

type Room struct {
	Room            *int64  `json:"room,omitempty"`
	Description     *string `json:"description,omitempty"`
	PinRequired     *bool   `json:"pin_required,omitempty"`
	MaxPublishers   *int    `json:"max_publishers,omitempty"`
	Bitrate         *int    `json:"bitrate,omitempty"`
	BitrateCap      *bool   `json:"bitrate_cap,omitempty"`
	FirFreq         *int    `json:"fir_freq,omitempty"`
	AudioCodec      *string `json:"audiocodec,omitempty"`
	VideoCodec      *string `json:"videocodec,omitempty"`
	Record          *bool   `json:"record,omitempty"`
	RecordDir       *string `json:"record_dir,omitempty"`
	LockRecord      *bool   `json:"lock_record,omitempty"`
	NumParticipants *int    `json:"num_participants,omitempty"`
	SamplingRate    *int    `json:"sampling_rate,omitempty"`
	ID              *int64  `json:"id,omitempty"`
	Name            *string `json:"name,omitempty"`
	Type            *string `json:"type,omitempty"`
	Metadata        *string `json:"metadata,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
	AudioAgeMs      *int    `json:"audio_age_ms,omitempty"`
	VideoAgeMs      *int    `json:"video_age_ms,omitempty"`
	Pin             *string `json:"pin,omitempty"`
	IsPrivate       *string `json:"is_private,omitempty"`
	Viewers         *int    `json:"viewers,omitempty"`
	Audio           *bool   `json:"audio,omitempty"`
	AudioPort       *int    `json:"audioport,omitempty"`
	AudioRTCPPort   *int    `json:"audiortcpport,omitempty"`
	AudioPt         *int    `json:"audiopt,omitempty"`
	AudioRTPMap     *string `json:"audiortpmap,omitempty"`
	AudioFMTP       *string `json:"audiofmtp,omitempty"`
	Video           *bool   `json:"video,omitempty"`
	VideoPort       *int    `json:"videoport,omitempty"`
	VideoRTCPPort   *int    `json:"videortcpport,omitempty"`
	VideoPt         *int    `json:"videopt,omitempty"`
	VideoRTPMap     *string `json:"videortpmap,omitempty"`
	VideoFMTP       *int    `json:"videofmtp,omitempty"`
}

type RTPForwarders struct {
	PublisherID  *int64          `json:"publisher_id,omitempty"`
	RTPForwarder *[]RTPForwarder `json:"rtp_forwarder,omitempty"`
	StreamID     *int64          `json:"stream_id,omitempty"`
	IP           *string         `json:"ip,omitempty"`
	Port         *int            `json:"port,omitempty"`
	SSRC         *int64          `json:"ssrc,omitempty"`
	Codec        *string         `json:"codec,omitempty"`
	PType        *int64          `json:"ptype,omitempty"`
	SRTP         *bool           `json:"srtp,omitempty"`
	AlwaysOn     *bool           `json:"always_on,omitempty"`
}

type RTPForwarder struct {
	AudioStreamID *int64  `json:"audio_stream_id,omitempty"`
	VideoStreamID *int64  `json:"video_stream_id,omitempty"`
	DataStreamID  *int64  `json:"data_stream_id,omitempty"`
	IP            *string `json:"ip,omitempty"`
	Port          *int    `json:"port,omitempty"`
	RTCPPort      *int    `json:"rtcp_port,omitempty"`
	SSRC          *int64  `json:"ssrc,omitempty"`
	Pt            *int64  `json:"pt,omitempty"`
	SubStream     *string `json:"substream,omitempty"`
	SRTP          *bool   `json:"srtp,omitempty"`
}
