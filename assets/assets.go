package assets

import (
	_ "embed"
)

//go:embed notification.wav
var NotificationWav []byte

//go:embed errorNotification.wav
var ErrorNotificationWav []byte

//go:embed silence.wav
var SilenceWav []byte

//go:embed startup.wav
var StartupWav []byte