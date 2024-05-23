package qrseq

import (
	"errors"
	"image"

	"github.com/airsigner/qrseq/internal"
)

type ChunkSize uint16

const (
	ChunkSizeUnknown ChunkSize = 0
	ChunkSize32      ChunkSize = ChunkSize(internal.ChunkSize32)
	ChunkSize64      ChunkSize = ChunkSize(internal.ChunkSize64)
	ChunkSize128     ChunkSize = ChunkSize(internal.ChunkSize128)
	ChunkSize256     ChunkSize = ChunkSize(internal.ChunkSize256)
	ChunkSize512     ChunkSize = ChunkSize(internal.ChunkSize512)
	ChunkSize1024    ChunkSize = ChunkSize(internal.ChunkSize1024)
)

type QRSequence struct {
	ChunkSize  ChunkSize
	chunks     []*internal.QRChunk
	nrReceived int
}

// New creates a new QRSequence with the given data and chunk size.
//
// Parameters:
// - data: a byte slice containing the data to be split into chunks.
// - chunkSize: a ChunkSize enum value specifying the size of each chunk.
//
// Returns:
// - a pointer to a QRSequence object.
func New(data []byte, chunkSize ChunkSize) *QRSequence {
	s := new(QRSequence)
	s.ChunkSize = ChunkSize(chunkSize)
	s.chunks = internal.CreateChunks(data, uint16(chunkSize))
	s.nrReceived = len(s.chunks)
	return s
}

// NewEmpty creates a new QRSequence with an unknown chunk size and an empty
// slice of QRChunks.
//
// An empty sequence should be used to start decoding chunked qr images into the
// sequence.
//
// Returns:
// - a pointer to a QRSequence object.
func NewEmpty() *QRSequence {
	return &QRSequence{
		ChunkSize: ChunkSizeUnknown,
		chunks:    make([]*internal.QRChunk, 0),
	}
}

// IsComplete checks if the QRSequence is complete.
//
// It returns true if all chunks of the sequence have been received, false
// otherwise.
//
// Returns:
// - bool: true if the QRSequence is complete, false otherwise.
func (s QRSequence) IsComplete() bool {
	if s.ChunkSize == ChunkSizeUnknown {
		return false
	}
	return s.nrReceived == len(s.chunks)
}

// Progress returns the progress of the QRSequence.
//
// It calculates the progress as a float32 value between 0 and 1.
// If all chunks have been received, it returns 1. Otherwise, it calculates the
// progress by dividing the number of received chunks by the total number of
// chunks.
//
// Returns:
// - float32: the progress of the QRSequence.
func (s QRSequence) Progress() float32 {
	if s.ChunkSize == ChunkSizeUnknown {
		return 0
	}
	if s.nrReceived == len(s.chunks) {
		return 1
	}
	return float32(s.nrReceived) / float32(len(s.chunks))
}

// Data returns the complete data of the QRSequence if it is complete, otherwise
// it returns nil.
//
// Returns:
// - []byte: the data of the QRSequence if it is complete, otherwise nil.
func (s QRSequence) Data() []byte {
	if !s.IsComplete() {
		return nil
	}
	return internal.GetData(s.chunks)
}

// QRCodes generates a slice of QR codes for each chunk in the QRSequence.
//
// It takes an integer parameter `blockSize` which specifies the size of the QR
// code blocks.
//
// Returns:
//   - []image.Image: a slice of QR codes generated for each chunk in the
//     QRSequence.
//   - error: an error if the QRSequence is not complete or if there is an error
//     while generating the QR codes.
func (s QRSequence) QRCodes(blockSize int) ([]image.Image, error) {
	if !s.IsComplete() {
		return nil, errors.New("sequence not complete")
	}

	images := make([]image.Image, 0, len(s.chunks))
	for _, chunk := range s.chunks {
		qr, err := chunk.QRCode(blockSize)
		if err != nil {
			return nil, err
		}
		images = append(images, qr)
	}

	return images, nil
}

// DecodeImage decodes an image into a QRSequence.
//
// It takes an image.Image as a parameter and attempts to decode it into a
// QRChunk.
// If the QRSequence is already complete, it returns nil.
// If the decoding is successful, the chunk is added to the QRSequence and nil
// is returned.
// If there is an error during decoding, the error is returned.
//
// Parameters:
// - img: an image.Image to be decoded into a QRChunk.
//
// Returns:
//   - error: an error if there was an issue decoding the image or if the
//     QRSequence is already complete.
func (s *QRSequence) DecodeImage(img image.Image) error {
	if s.IsComplete() {
		return nil
	}

	chunk, err := internal.NewChunkFromImage(img)
	if err != nil {
		return err
	}

	s.addChunk(chunk)
	return nil
}

// AddChunkFromBytes adds a chunk of data to the QRSequence.
//
// It takes a byte slice as a parameter, which represents the data to be added.
// If the QRSequence is already complete, the function returns immediately.
// It creates a new chunk from the given data using the internal.NewChunk
// function.
// If the chunk is nil, the function returns.
// Otherwise, it adds the chunk to the QRSequence using the addChunk method.
func (s *QRSequence) AddChunkFromBytes(data []byte) {
	if s.IsComplete() {
		return
	}

	chunk := internal.NewChunk(data)
	if chunk == nil {
		return
	}

	s.addChunk(chunk)
}

// addChunk adds a chunk of data to the QRSequence.
//
// It takes a pointer to a QRChunk as a parameter, which represents the data to
// be added.
// If the QRSequence is already complete, the function returns immediately.
// If the ChunkSize is unknown, it sets the ChunkSize to the size of the given
// chunk and creates a slice of QRChunks with the total size.
// If the chunk with the same number already exists in the QRSequence, the
// function returns.
// Otherwise, it adds the chunk to the QRSequence and increments the number of
// received chunks.
//
// Parameters:
// - chunk: a pointer to a QRChunk representing the data to be added.
func (s *QRSequence) addChunk(chunk *internal.QRChunk) {
	if s.ChunkSize == ChunkSizeUnknown {
		s.ChunkSize = ChunkSize(chunk.Size())
		s.chunks = make([]*internal.QRChunk, chunk.Tot())
		s.nrReceived = 0
	}

	if s.chunks[chunk.Nr()] == nil {
		s.chunks[chunk.Nr()] = chunk
		s.nrReceived++
	}
}
