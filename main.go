package escpos

import (
	"errors"
	"fmt"
	"github.com/qiniu/iconv"
	"image"
	"io"
	"math"
)
type Style struct {
	Bold			bool
	Width, Height	uint8
	Reverse			bool
	Underline		uint8 // can be 0, 1 or 2
	UpsideDown		bool
	Rotate			bool
	Justify			uint8
}

const (
	JustifyLeft uint8 = 0
	JustifyCenter uint8 = 1
	JustifyRight uint8 = 2
	QRCodeErrorCorrectionLevelL uint8 = 48
	QRCodeErrorCorrectionLevelM uint8 = 49
	QRCodeErrorCorrectionLevelQ uint8 = 50
	QRCodeErrorCorrectionLevelH uint8 = 51
	esc byte = 0x1B
	gs byte = 0x1D
	fs byte = 0x1C
)

type Escpos struct {
	dst   io.Writer
	Style Style
}

// New create an Escpos printer
func New(dst io.Writer) (e *Escpos) {
	e = &Escpos{dst: dst}
	return
}

// WriteRaw write raw bytes to the printer
func (e *Escpos) WriteRaw(data []byte) (int, error) {
	if len(data) > 0 {
		return e.dst.Write(data)
	}
	return 0, nil
}

// Stuff for writing text.

// Writes a string using the predefined options.
func (e *Escpos) Write(data string) (int, error) {
	// we gonna write sum text, so apply the styles!
	var err error
	// Bold
	_, err = e.WriteRaw([]byte{esc, 'E', boolToByte(e.Style.Bold)})
	if err != nil {
		// return 0 written bytes here, because technically we did not write any of the bytes of data
		return 0, err
	}
	// Underline
	_, err = e.WriteRaw([]byte{esc, '-', e.Style.Underline})
	if err != nil {
		return 0, err
	}
	// Reverse
	_, err = e.WriteRaw([]byte{gs, 'B', boolToByte(e.Style.Reverse)})
	if err != nil {
		return 0, err
	}

	// Rotate
	_, err = e.WriteRaw([]byte{esc, 'V', boolToByte(e.Style.Rotate)})
	if err != nil {
		return 0, err
	}

	// UpsideDown
	_, err = e.WriteRaw([]byte{esc, '{', boolToByte(e.Style.UpsideDown)})
	if err != nil {
		return 0, err
	}
	// Justify
	_, err = e.WriteRaw([]byte{esc, 'a', e.Style.Justify})
	if err != nil {
		return 0, err
	}

	// Width / Height
	_, err = e.WriteRaw([]byte{gs, '!', ((e.Style.Width - 1) << 4) | (e.Style.Height - 1)})
	if err != nil {
		return 0, err
	}


	return e.WriteRaw([]byte(data))
}

// WriteGBK writes a string to the printer using GBK encoding
func (e *Escpos) WriteGBK(data string) (int, error) {
	cd, err := iconv.Open("gbk", "utf-8")
	if err != nil {
		return 0, err
	}
	defer cd.Close()
	gbk := cd.ConvString(data)
	return e.Write(gbk)
}

// WriteWEU writes a string to the printer using Western European encoding
func (e *Escpos) WriteWEU(data string) (int, error) {
	cd, err := iconv.Open("cp850", "utf-8")
	if err != nil {
		return 0, err
	}
	defer cd.Close()
	weu := cd.ConvString(data)
	return e.Write(weu)
}

// Sets the printer to print Bold text.
func (e *Escpos) Bold(p bool) *Escpos {
	e.Style.Bold = p
	return e
}

// Sets the Underline. p can be 0, 1 or 2. It defines the thickness of the underline in dots
func (e *Escpos) Underline(p uint8) *Escpos {
	e.Style.Underline = p
	return e
}

// Sets Reverse printing. If true the printer will inverse to white text on black background.
func (e *Escpos) Reverse(p bool) *Escpos {
	e.Style.Reverse = p
	return e
}

// Sets the justification of the text. Possible values are 0, 1 or 2. You can use
// JustifyLeft for left alignment
// JustifyCenter for center alignment
// JustifyRight for right alignment
func (e *Escpos) Justify(p uint8) *Escpos {
	e.Style.Justify = p
	return e
}

// Toggles 90° CW rotation
func (e *Escpos) Rotate(p bool) *Escpos {
	e.Style.Rotate = p
	return e
}

// Toggles UpsideDown printing
func (e *Escpos) UpsideDown(p bool) *Escpos {
	e.Style.UpsideDown = p
	return e
}

// Sets the size of the font. Width and Height should be between 0 and 5. If the value is bigger than 5, 5 is used.
func (e *Escpos) Size(width uint8, height uint8) *Escpos {
	// Values > 5 are not supported by esc/pos, so we'll set 5 as the maximum.
	if width > 5 {
		width = 5
	}
	if height > 5 {
		height = 5
	}
	e.Style.Width = width
	e.Style.Height = height
	return e
}


// Barcode stuff.

// Sets the position of the HRI characters
// 0: Not Printed
// 1: Above the bar code
// 2: Below the bar code
// 3: Both
func (e *Escpos) HRIPosition(p uint8) (int, error) {
	if p > 3 {
		p = 0
	}
	return e.WriteRaw([]byte{gs, 'H', p})
}
// Sets the HRI font to either
// false: Font A (12x24) or
// true: Font B (9x24)
func (e *Escpos) HRIFont(p bool) (int, error) {
	return e.WriteRaw([]byte{gs, 'f', boolToByte(p)})
}

// Sets the height for a bar code. Default is 162.
func (e *Escpos) BarcodeHeight(p uint8) (int, error) {
	return e.WriteRaw([]byte{gs, 'h', p})
}

// Sets the horizontal size for a bar code. Default is 3. Must be between 2 and 6
func (e *Escpos) BarcodeWidth(p uint8) (int, error) {
	if p < 2 {
		p = 2
	}
	if p > 6 {
		p = 6
	}
	return e.WriteRaw([]byte{gs, 'h', p})
}

// GS k for printing barcode

func (e *Escpos) TestBarcode() (int, error) {
	return e.WriteRaw([]byte{gs, 'k', 0, '1', '2', '1', '1', '4', '5', '6', '5', '4', '3', '4', '5', 0})
}

// Prints a UPCA Barcode. code can only be numerical characters and must have a length of 11 or 12
func (e *Escpos) UPCA(code string) (int, error) {
	if len(code) != 11 && len(code) != 12 {
		return 0, errors.New("code should have a length between 11 and 12")
	}
	if !onlyDigits(code) {
		return 0, errors.New("code can only contain numerical characters")
	}
	byteCode := append([]byte(code), 0)
	return e.WriteRaw(append([]byte{gs, 'k', 0}, byteCode...))
}

// Prints a UPCE Barcode. code can only be numerical characters and must have a length of 11 or 12
func (e *Escpos) UPCE(code string) (int, error) {
	if len(code) != 11 && len(code) != 12 {
		return 0, errors.New("code should have a length between 11 and 12")
	}
	if !onlyDigits(code) {
		return 0, errors.New("code can only contain numerical characters")
	}
	byteCode := append([]byte(code), 0)
	return e.WriteRaw(append([]byte{gs, 'k', 1}, byteCode...))
}

// Prints a EAN13 Barcode. code can only be numerical characters and must have a length of 12 or 13
func (e *Escpos) EAN13(code string) (int, error) {
	if len(code) != 12 && len(code) != 13 {
		return 0, errors.New("code should have a length between 12 and 13")
	}
	if !onlyDigits(code) {
		return 0, errors.New("code can only contain numerical characters")
	}
	byteCode := append([]byte(code), 0)
	return e.WriteRaw(append([]byte{gs, 'k', 2}, byteCode...))
}

// Prints a EAN8 Barcode. code can only be numerical characters and must have a length of 7 or 8
func (e *Escpos) EAN8(code string) (int, error) {
	if len(code) != 7 && len(code) != 8 {
		return 0, errors.New("code should have a length between 7 and 8")
	}
	if !onlyDigits(code) {
		return 0, errors.New("code can only contain numerical characters")
	}
	byteCode := append([]byte(code), 0)
	return e.WriteRaw(append([]byte{gs, 'k', 3}, byteCode...))
}

// TODO:
// CODE39, ITF, CODABAR

// Prints a QR Code.
// code specifies the data to be printed
// model specifies the qr code model. false for model 1, true for model 2
// size specifies the size in dots
func (e *Escpos) QRCode(code string, model bool, size uint8, correctionLevel uint8) (int, error) {
	if len(code) > 7089 {
		return 0, errors.New("the code is too long, it's length should be smaller than 7090")
	}
	var m byte = 49
	var err error
	// set the qr code model
	if model {
		m = 50
	}
	_, err = e.WriteRaw([]byte{gs, '(', 'k', 4, 0, 49, 65, m, 0})
	if err != nil {
		return 0, err
	}

	// set the qr code size
	_, err = e.WriteRaw([]byte{gs, '(', 'k', 3, 0, 49, 67, size})
	if err != nil {
		return 0, err
	}

	// set the qr code error correction level
	if correctionLevel < 48 {
		correctionLevel = 48
	}
	if correctionLevel > 51 {
		correctionLevel = 51
	}
	_, err = e.WriteRaw([]byte{gs, '(', 'k', 3, 0, 49, 69, size})
	if err != nil {
		return 0, err
	}

	// store the data in the buffer
	// we now write stuff to the printer, so lets save it for returning

	// pL and pH define the size of the data. Data ranges from 1 to (pL + pH*256)-3
	// 3 < pL + pH*256 < 7093
	var codeLength = len(code)+3
	var pL, pH byte
	pH = byte(int(math.Floor(float64(codeLength) / 256)))
	pL = byte(codeLength - 256*int(pH))
	fmt.Printf("%d %d", pH, pL)

	written, err := e.WriteRaw(append([]byte{gs, '(', 'k', pL, pH, 49, 80, 48}, []byte(code)...))
	if err != nil {
		return written, err
	}

	// finally print the buffer
	_, err = e.WriteRaw([]byte{gs, '(', 'k', 3, 0, 49, 81, 48})
	if err != nil {
		return written, err
	}

	return written, nil
}

// todo PDF417
//func (e *Escpos) PDF417() (int, error) {
//
//}

// Image stuff.
// todo.

// Prints an image
func (e *Escpos) PrintImage(image image.Image) (int, error) {
	xL, xH, yL, yH, data := printImage(image)
	return e.WriteRaw(append([]byte{gs, 'v', 48, 0, xL, xH, yL, yH}, data...))
}

// Print a predefined bit image with index p and mode mode
func (e *Escpos) PrintNVBitImage(p uint8, mode uint8) (int, error) {
	if p == 0 {
		return 0, errors.New("start index of nv bit images start at 1")
	}
	if mode > 3 {
		return 0, errors.New("mode only supports values from 0 to 3")
	}

	return e.WriteRaw([]byte{fs, 'd', p, mode})
}

// Configuration stuff

// Sends a newline to the printer.
func (e *Escpos) LineFeed() (int, error) {
	return e.Write("\n")
}
// According to command manual this prints and feeds the paper p*line spacing.
func (e *Escpos) LineFeedD(p uint8) (int, error) {
	return e.WriteRaw([]byte{esc, 'd', p})
}

// Sets the line spacing to the default. According to command manual this is 1/6 inch
func (e *Escpos) DefaultLineSpacing() (int, error) {
	return e.WriteRaw([]byte{esc, '2'})
}

// Sets the line spacing to multiples of the "horizontal and vertical motion units".. Those can be set with MotionUnits
func (e *Escpos) LineSpacing(p uint8) (int, error) {
	return e.WriteRaw([]byte{esc, '3', p})
}
// Initializes the printer to the settings it had when turned on
func (e *Escpos) Initialize() (int, error) {
	return e.WriteRaw([]byte{esc, '@'})
}

// Sets the horizontal (x) and vertical (y) motion units to 1/x inch and 1/y inch. Well... According to the manual anyway. You may not want to use this, as it does not seem to do the same on an Epson TM-20II
func (e *Escpos) MotionUnits(x, y uint8) (int, error) {
	return e.WriteRaw([]byte{gs, 'P', x, y})
}

// Feeds the paper to the end and performs a Cut. In the ESC/POS Command Manual there is also PartialCut and FullCut documented, but it does exactly the same.
func (e *Escpos) Cut() (int, error) {
	return e.WriteRaw([]byte{gs, 'V', 'A', '0'})
}




// Helpers
func boolToByte(b bool) byte {
	if b {
		return '1'
	}
	return '0'
}
func onlyDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}