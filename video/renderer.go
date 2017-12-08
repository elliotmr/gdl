package video

import "github.com/pkg/errors"

const (
	RendererSoftware      = 1 << iota // The renderer is a software fallback
	RendererAccelerated               // The renderer uses hardware acceleration
	RendererPresentVSync              // Present is synchronized with the refresh rate
	RendererTargetRexture             // The renderer supports rendering to texture
)

type RendererInfo struct {
	name                              string
	flags                             uint32
	numTextureFormats                 uint32
	textureFormats                    [16]uint32
	maxTextureWidth, maxTextureHeight int
}

type Renderer struct {
	info RendererInfo

	// Window associated with renderer
	window *Window
	hidden bool

	// Logical Resolution for Rendering
	logicalW, logicalH             int
	logicalWBackup, logicalHBackup int

	integerScale bool

	viewport       Rect
	viewportBackup Rect

	clippingEnabled       bool
	clippingEnabledBackup bool

	scale       Point
	scaleBackup Point

	textures *Texture
	target   *Texture

	r, g, b, a uint8
	blendMode  uint32

	driverData interface{}
}

type RenderDriver interface {
	CreateRenderer(window *Window, flags uint32) (*Renderer, error)
	Info() RendererInfo
}

func CreateRenderer(window *Window, flags uint32) (*Renderer, error) {
	if window  == nil {
		return nil, errors.New("invalid window")
	}
	return nil, nil
}
