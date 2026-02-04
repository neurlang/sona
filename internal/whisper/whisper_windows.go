//go:build windows

package whisper

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/include
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/lib
#cgo LDFLAGS: -lwhisper -lggml -lggml-base -lggml-cpu -lggml-vulkan
#cgo LDFLAGS: -lvulkan-1 -lm
#cgo LDFLAGS: -Wl,-Bstatic -lstdc++ -lgomp -lwinpthread -Wl,-Bdynamic
*/
import "C"
