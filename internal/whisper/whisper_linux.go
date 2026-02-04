//go:build linux

package whisper

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/include
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/lib
#cgo LDFLAGS: -lwhisper -lggml -lggml-base -lggml-cpu -lggml-vulkan
#cgo LDFLAGS: -lvulkan -lstdc++ -lm -lpthread -lgomp
*/
import "C"
