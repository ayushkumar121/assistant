package assets

import (
	_ "embed"
)

//go:embed notification.wav
var NotificationWav []byte

//go:embed silence.wav
var SilenceWav []byte
