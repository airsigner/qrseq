package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"image"

	"github.com/makiuchi-d/gozxing"
	qrzxing "github.com/makiuchi-d/gozxing/qrcode"
	"github.com/yeqown/go-qrcode/v2"
)

const (
	ChunkSize32   uint16 = 32
	ChunkSize64   uint16 = 64
	ChunkSize128  uint16 = 128
	ChunkSize256  uint16 = 256
	ChunkSize512  uint16 = 512
	ChunkSize1024 uint16 = 1024
)

func isValidChunkSize(cs uint16) bool {
	switch cs {
	case ChunkSize32, ChunkSize64, ChunkSize128, ChunkSize256, ChunkSize512, ChunkSize1024:
		return true
	default:
		return false
	}
}

type QRChunk struct {
	nr   uint8  // chunk number
	tot  uint8  // total number of chunks
	cs   uint16 // chunk size in bytes (data is chunksize - 4 bytes (nr, tot, cs))
	data []byte
}

// NewChunk creates a new QRChunk from the given byte slice.
//
// The function takes a byte slice as input and extracts the necessary
// information to create a new QRChunk. It first extracts the values for nr
// and tot from the first two bytes of the input data. Then, it reads the
// chunk size from the next two bytes and checks if it is a valid chunk size
// using the isValidChunkSize function. If the chunk size is invalid, the
// function returns nil. Otherwise, it creates a new QRChunk with the
// extracted values and the remaining data.
//
// Parameters:
//   - data: a byte slice containing the data for the QRChunk.
//
// Returns:
//   - *QRChunk: a pointer to the newly created QRChunk, or nil if the chunk
//     size is invalid.
func NewChunk(data []byte) *QRChunk {
	nr := uint8(data[0])
	tot := uint8(data[1])

	cs := uint16(0)
	csBuf := bytes.NewReader(data[2:4])
	err := binary.Read(csBuf, binary.LittleEndian, &cs)
	if err != nil {
		return nil
	}
	if !isValidChunkSize(cs) {
		return nil
	}

	return &QRChunk{
		nr:  nr,
		tot: tot,
		cs:  cs,
		data: func() []byte {
			if len(data) > int(cs) {
				return data[4:cs]
			}
			return data[4:]
		}(),
	}
}

// NewChunkFromImage decodes an image into a QRChunk.
//
// It takes an image.Image as a parameter and attempts to decode it into a
// QRChunk. It first creates a BinaryBitmap from the image using the
// gozxing.NewBinaryBitmapFromImage function. Then it creates a QRCodeReader
// and uses it to decode the BinaryBitmap into a QRCodeData object. The
// QRCodeData object contains the text of the QR code, which is then decoded
// from base64 to bytes using the base64.StdEncoding.DecodeString function.
// Finally, it creates a new QRChunk using the NewChunk function and returns it
// along with any error that occurred during the decoding process. If the
// decoded chunk is invalid, it returns an error.
//
// Parameters:
// - img: an image.Image to be decoded into a QRChunk.
//
// Returns:
//   - *QRChunk: the decoded QRChunk.
//   - error: an error if there was an issue decoding the image or if the
//     decoded chunk is invalid.
func NewChunkFromImage(img image.Image) (*QRChunk, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, err
	}

	reader := qrzxing.NewQRCodeReader()
	data, err := reader.Decode(bmp, nil)
	if err != nil {
		return nil, err
	}

	bytes, err := base64.StdEncoding.DecodeString(data.GetText())
	if err != nil {
		return nil, err
	}

	chunk := NewChunk(bytes)
	if chunk == nil {
		return nil, errors.New("invalid chunk")
	}
	return chunk, nil
}

// CreateChunks generates a slice of QRChunk pointers based on the given data
// and chunk size.
//
// Parameters:
// - data: a byte slice containing the data to be split into chunks.
// - chunkSize: an unsigned 16-bit integer specifying the size of each chunk.
//
// Returns:
// - []*QRChunk: a slice of pointers to QRChunk objects.
func CreateChunks(data []byte, chunkSize uint16) []*QRChunk {
	ds := chunkSize - 4
	tot := len(data) / int(ds)
	if len(data)%int(ds) != 0 {
		tot++
	}
	chunks := make([]*QRChunk, 0, tot)

	for i := 0; i < tot; i++ {
		chunks = append(chunks, &QRChunk{
			nr:  uint8(i),
			tot: uint8(tot),
			cs:  chunkSize,
			data: func(i int) []byte {
				s := i * int(ds)
				e := s + int(ds)
				if e > len(data) {
					return data[s:]
				}
				return data[s:e]
			}(i),
		})
	}
	return chunks
}

// GetData generates a byte slice containing the data from the given slice of
// QRChunk pointers.
//
// It takes a slice of pointers to QRChunk objects as a parameter. If the slice
// is empty, it returns nil.
// Otherwise, it creates a new byte slice with an initial capacity equal to the
// estimated data size of the first chunk.
// It then iterates over each chunk in the slice and appends its data to the
// data slice.
// Finally, it returns the generated data slice.
//
// Parameters:
// - chunks: a slice of pointers to QRChunk objects.
//
// Returns:
//   - []byte: a byte slice containing the data from the QRChunk pointers, or
//     nil if the input slice is empty.
func GetData(chunks []*QRChunk) []byte {
	if len(chunks) == 0 {
		return nil
	}

	data := make([]byte, 0, chunks[0].estimatedDataSize())
	for _, chunk := range chunks {
		data = append(data, chunk.data...)
	}
	return data
}

// Nr returns sequence number of this qr chunk.
func (c QRChunk) Nr() uint8 {
	return c.nr
}

// Tot returns the total number of qr chunks in this sequence.
func (c QRChunk) Tot() uint8 {
	return c.tot
}

// Size returns the chunksize
func (c QRChunk) Size() uint16 {
	return c.cs
}

// Data returns the payload data
func (c QRChunk) Data() []byte {
	return c.data
}

// QRCode generates a QR code image based on the data of the QRChunk.
//
// It takes an integer parameter `blockSize` which represents the size of the
// blocks in the QR code.
// The function returns two values: `img` of type `image.Image` which is the
// generated QR code image,
// and `err` of type `error` which indicates any error that occurred during the
// generation process.
//
// If the `blockSize` parameter is less than 1, the function returns an error
// indicating an invalid block size.
// The function then creates a new QR code using the `qrcode.New` function,
// passing the base64-encoded data of the QRChunk.
// If there is an error creating the QR code, the function returns the error.
// The function creates a new `ImageWriter` with a callback function that
// assigns the generated image to the `img` variable.
// The `ImageWriter` is configured with the `Padding` and `BlockSize` options
// set to the `blockSize` parameter.
// The function saves the QR code using the `qr.Save` method, passing the
// `ImageWriter` as the writer.
// If there is an error saving the QR code, the function returns the error.
// The function returns the generated image and any error that occurred during
// the process.
func (c QRChunk) QRCode(blockSize int) (img image.Image, err error) {
	if blockSize < 1 {
		err = errors.New("invalid block size")
		return
	}

	qrdata := make([]byte, 0, len(c.data)+4)
	qrdata = append(qrdata, c.nr)
	qrdata = append(qrdata, c.tot)

	csBuff := new(bytes.Buffer)
	err = binary.Write(csBuff, binary.LittleEndian, c.cs)
	if err != nil {
		return
	}
	qrdata = append(qrdata, csBuff.Bytes()...)

	qrdata = append(qrdata, c.data...)

	qr, err := qrcode.New(base64.StdEncoding.EncodeToString(qrdata))
	if err != nil {
		return
	}

	w := NewImageWriter(
		func(res image.Image) {
			img = res
		}, &Option{
			Padding:   blockSize,
			BlockSize: blockSize,
		})

	if err = qr.Save(w); err != nil {
		return
	}
	return
}

func (c QRChunk) estimatedDataSize() uint64 {
	return uint64(c.cs-4) * uint64(c.tot)
}
