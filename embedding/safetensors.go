package embedding

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// Safetensors is a plain data format (JSON header + raw tensor bytes), so
// loading a model can never execute code, unlike pickle-based checkpoints.

type tensor struct {
	Shape []int
	Data  []float32
}

type tensorInfo struct {
	Dtype   string `json:"dtype"`
	Shape   []int  `json:"shape"`
	Offsets [2]int `json:"data_offsets"`
}

func readSafetensors(path string) (map[string]tensor, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < 8 {
		return nil, fmt.Errorf("%s: file too small", path)
	}
	headerLen := binary.LittleEndian.Uint64(raw[:8])
	if headerLen > uint64(len(raw)-8) {
		return nil, fmt.Errorf("%s: invalid header length", path)
	}

	var header map[string]json.RawMessage
	if err := json.Unmarshal(raw[8:8+headerLen], &header); err != nil {
		return nil, fmt.Errorf("%s: invalid header: %v", path, err)
	}
	body := raw[8+headerLen:]

	tensors := make(map[string]tensor)
	for name, msg := range header {
		if name == "__metadata__" {
			continue
		}
		var info tensorInfo
		if err := json.Unmarshal(msg, &info); err != nil {
			return nil, fmt.Errorf("%s: tensor %s: %v", path, name, err)
		}

		count := 1
		for _, d := range info.Shape {
			count *= d
		}
		start, end := info.Offsets[0], info.Offsets[1]
		if start < 0 || end > len(body) || start > end {
			return nil, fmt.Errorf("%s: tensor %s: offsets out of range", path, name)
		}
		b := body[start:end]

		data := make([]float32, count)
		switch info.Dtype {
		case "F32":
			if len(b) != count*4 {
				return nil, fmt.Errorf("%s: tensor %s: size mismatch", path, name)
			}
			for i := range data {
				data[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
			}
		case "F16":
			if len(b) != count*2 {
				return nil, fmt.Errorf("%s: tensor %s: size mismatch", path, name)
			}
			for i := range data {
				data[i] = halfToFloat(binary.LittleEndian.Uint16(b[i*2:]))
			}
		default:
			// Integer buffers such as position_ids are not needed for inference
			continue
		}
		tensors[name] = tensor{Shape: info.Shape, Data: data}
	}

	return tensors, nil
}

// ConvertToF16 rewrites a safetensors file with float16 weights, halving its
// size, and keeps only the tensors for which keep returns true.
func ConvertToF16(srcPath, dstPath string, keep func(name string) bool) error {
	tensors, err := readSafetensors(srcPath)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(tensors))
	for name := range tensors {
		if keep(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	header := make(map[string]tensorInfo, len(names))
	var body []byte
	for _, name := range names {
		t := tensors[name]
		start := len(body)
		for _, v := range t.Data {
			body = binary.LittleEndian.AppendUint16(body, floatToHalf(v))
		}
		header[name] = tensorInfo{Dtype: "F16", Shape: t.Shape, Offsets: [2]int{start, len(body)}}
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return err
	}
	// Pad the header so tensor data starts on an 8-byte boundary
	for len(headerJSON)%8 != 0 {
		headerJSON = append(headerJSON, ' ')
	}

	out := binary.LittleEndian.AppendUint64(nil, uint64(len(headerJSON)))
	out = append(out, headerJSON...)
	out = append(out, body...)
	return os.WriteFile(dstPath, out, 0o644)
}

func halfToFloat(h uint16) float32 {
	sign := uint32(h&0x8000) << 16
	exp := uint32(h>>10) & 0x1f
	mant := uint32(h & 0x3ff)

	switch exp {
	case 0:
		// Zero or subnormal: mant * 2^-24
		f := float32(mant) / (1 << 24)
		if sign != 0 {
			f = -f
		}
		return f
	case 0x1f:
		return math.Float32frombits(sign | 0x7f800000 | mant<<13)
	}
	return math.Float32frombits(sign | (exp+112)<<23 | mant<<13)
}

// floatToHalf converts with round-to-nearest-even.
func floatToHalf(f float32) uint16 {
	b := math.Float32bits(f)
	sign := uint16(b>>16) & 0x8000
	rawExp := int(b>>23) & 0xff
	mant := b & 0x7fffff

	if rawExp == 0xff {
		if mant != 0 {
			return sign | 0x7e00
		}
		return sign | 0x7c00
	}

	exp := rawExp - 127 + 15
	if exp >= 0x1f {
		return sign | 0x7c00
	}
	if exp <= 0 {
		if exp < -10 {
			return sign
		}
		mant |= 0x800000
		shift := uint(14 - exp)
		half := mant >> shift
		rem := mant & (1<<shift - 1)
		halfway := uint32(1) << (shift - 1)
		if rem > halfway || (rem == halfway && half&1 == 1) {
			half++
		}
		return sign | uint16(half)
	}

	half := uint32(exp)<<10 | mant>>13
	rem := mant & 0x1fff
	if rem > 0x1000 || (rem == 0x1000 && half&1 == 1) {
		half++
	}
	return sign | uint16(half)
}
