//go:build darwin

package whisper

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/include
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/lib
#cgo LDFLAGS: -lwhisper -lggml -lggml-base -lggml-cpu -lggml-metal -lggml-blas
#cgo LDFLAGS: -framework Accelerate -framework Metal -framework Foundation -framework MetalKit -framework CoreGraphics
#cgo LDFLAGS: -lstdc++
*/
import "C"
