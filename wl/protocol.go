package wl

import (
	"encoding/binary"
	"fmt"
	"os"
    "encoding/hex"
)

const (
    DisplayErrorInvalidObject = 0 // server couldn't find object
    DisplayErrorInvalidMethod = 1 // method doesn't exist on the specified interface
    DisplayErrorNoMemory = 2 // server is out of memory
)

const (
    opCodeDisplayError = 0
    opCodeDisplayDeleteID = 1
)

const (
	opCodeDisplaySync = 0
	opCodeDisplayGetRegistry = 1
)

// Display Events
//
// Error
// The error event is sent out when a fatal (non-recoverable)
// error has occurred.  The object_id argument is the object
// where the error occurred, most often in response to a request
// to that object.  The code identifies the error and is defined
// by the object interface.  As such, each interface defines its
// own set of error codes.  The message is a brief description
// of the error, for (debugging) convenience.
//
// DeleteID
// This event is used internally by the object ID management
// logic.  When a client deletes an object, the server will send
// this event to acknowledge that it has seen the delete request.
// When the client receives this event, it will know that it can
// safely reuse the object ID.
type DisplayListener interface {
    Error(objectID uint32, code uint32, message string)
    DeleteID(id uint32)
}

// The core global object.  This is a special singleton object.  It
// is used for internal Wayland protocol features.
type Display struct {
    id ObjectID
    l  DisplayListener
	c  *Client
}

func (c *Client) NewDisplay() *Display {
	o := &Display{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Display) ID() ObjectID {
	return this.id
}

func (this *Display) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeDisplayError:
		if this.l == nil {
			fmt.Println("ignoring Error event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			objectID := hostByteOrder.Uint32(payload[0+off:4+off])
			code := hostByteOrder.Uint32(payload[4+off:8+off])
			len = int(hostByteOrder.Uint32(payload[8+off:12+off]))
			message := string(payload[12+off:12+len+off-1])
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}

			this.l.Error(objectID, code, message)
        }
	case opCodeDisplayDeleteID:
		if this.l == nil {
			fmt.Println("ignoring DeleteID event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			id := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.DeleteID(id)
        }
	
    }
}

func (this *Display) SetListener(listener DisplayListener) {
    this.l = listener
}

// The sync request asks the server to emit the 'done' event
// on the returned wl_callback object.  Since requests are
// handled in-order and events are delivered in-order, this can
// be used as a barrier to ensure all previous requests and the
// resulting events have been handled.
// 
// The object returned by this request will be destroyed by the
// compositor after the callback is fired and as such the client must not
// attempt to use it after that point.
// 
// The callback_data passed in the callback is the event serial.
func (this *Display) Sync() (*Callback, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewCallback()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDisplaySync)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}
// This request creates a registry object that allows the client
// to list and bind the global objects available from the
// compositor.
func (this *Display) GetRegistry() (*Registry, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewRegistry()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDisplayGetRegistry)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}

const (
    opCodeRegistryGlobal = 0
    opCodeRegistryGlobalRemove = 1
)

const (
	opCodeRegistryBind = 0
)

// Registry Events
//
// Global
// Notify the client of global objects.
// 
// The event notifies the client that a global object with
// the given name is now available, and it implements the
// given version of the given interface.
//
// GlobalRemove
// Notify the client of removed global objects.
// 
// This event notifies the client that the global identified
// by name is no longer available.  If the client bound to
// the global using the bind request, the client should now
// destroy that object.
// 
// The object remains valid and requests to the object will be
// ignored until the client destroys it, to avoid races between
// the global going away and a client sending a request to it.
type RegistryListener interface {
    Global(name uint32, iface string, version uint32)
    GlobalRemove(name uint32)
}

// The singleton global registry object.  The server has a number of
// global objects that are available to all clients.  These objects
// typically represent an actual object in the server (for example,
// an input device) or they are singleton objects that provide
// extension functionality.
// 
// When a client creates a registry object, the registry object
// will emit a global event for each global currently in the
// registry.  Globals come and go as a result of device or
// monitor hotplugs, reconfiguration or other events, and the
// registry will send out global and global_remove events to
// keep the client up to date with the changes.  To mark the end
// of the initial burst of events, the client can use the
// wl_display.sync request immediately after calling
// wl_display.get_registry.
// 
// A client can bind to a global object by using the bind
// request.  This creates a client-side handle that lets the object
// emit events to the client and lets the client invoke requests on
// the object.
type Registry struct {
    id ObjectID
    l  RegistryListener
	c  *Client
}

func (c *Client) NewRegistry() *Registry {
	o := &Registry{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Registry) ID() ObjectID {
	return this.id
}

func (this *Registry) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeRegistryGlobal:
		if this.l == nil {
			fmt.Println("ignoring Global event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			name := hostByteOrder.Uint32(payload[0+off:4+off])
			len = int(hostByteOrder.Uint32(payload[4+off:8+off]))
			iface := string(payload[8+off:8+len+off-1])
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}
			version := hostByteOrder.Uint32(payload[8+off:12+off])

			this.l.Global(name, iface, version)
        }
	case opCodeRegistryGlobalRemove:
		if this.l == nil {
			fmt.Println("ignoring GlobalRemove event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			name := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.GlobalRemove(name)
        }
	
    }
}

func (this *Registry) SetListener(listener RegistryListener) {
    this.l = listener
}

// Binds a new, client-created object to the server using the
// specified name as the identifier.
func (this *Registry) Bind(name uint32, id ObjectID) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, name)
	binary.Write(this.c.buf, hostByteOrder, id)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeRegistryBind)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    opCodeCallbackDone = 0
)

const (
)

// Callback Events
//
// Done
// Notify the client when the related request is done.
type CallbackListener interface {
    Done(callbackData uint32)
}

// Clients can handle the 'done' event to get notified when
// the related request is done.
type Callback struct {
    id ObjectID
    l  CallbackListener
	c  *Client
}

func (c *Client) NewCallback() *Callback {
	o := &Callback{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Callback) ID() ObjectID {
	return this.id
}

func (this *Callback) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeCallbackDone:
		if this.l == nil {
			fmt.Println("ignoring Done event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			callbackData := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Done(callbackData)
        }
	
    }
}

func (this *Callback) SetListener(listener CallbackListener) {
    this.l = listener
}


const (
)

const (
	opCodeCompositorCreateSurface = 0
	opCodeCompositorCreateRegion = 1
)

// Compositor Events
type CompositorListener interface {
}

// A compositor.  This object is a singleton global.  The
// compositor is in charge of combining the contents of multiple
// surfaces into one displayable output.
type Compositor struct {
    id ObjectID
    l  CompositorListener
	c  *Client
}

func (c *Client) NewCompositor() *Compositor {
	o := &Compositor{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Compositor) ID() ObjectID {
	return this.id
}

func (this *Compositor) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	
    }
}

func (this *Compositor) SetListener(listener CompositorListener) {
    this.l = listener
}

// Ask the compositor to create a new surface.
func (this *Compositor) CreateSurface() (*Surface, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewSurface()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeCompositorCreateSurface)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}
// Ask the compositor to create a new region.
func (this *Compositor) CreateRegion() (*Region, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewRegion()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeCompositorCreateRegion)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}

const (
)

const (
	opCodeShmPoolCreateBuffer = 0
	opCodeShmPoolDestroy = 1
	opCodeShmPoolResize = 2
)

// ShmPool Events
type ShmPoolListener interface {
}

// The wl_shm_pool object encapsulates a piece of memory shared
// between the compositor and client.  Through the wl_shm_pool
// object, the client can allocate shared memory wl_buffer objects.
// All objects created through the same pool share the same
// underlying mapped memory. Reusing the mapped memory avoids the
// setup/teardown overhead and is useful when interactively resizing
// a surface or for many small buffers.
type ShmPool struct {
    id ObjectID
    l  ShmPoolListener
	c  *Client
}

func (c *Client) NewShmPool() *ShmPool {
	o := &ShmPool{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *ShmPool) ID() ObjectID {
	return this.id
}

func (this *ShmPool) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	
    }
}

func (this *ShmPool) SetListener(listener ShmPoolListener) {
    this.l = listener
}

// Create a wl_buffer object from the pool.
// 
// The buffer is created offset bytes into the pool and has
// width and height as specified.  The stride argument specifies
// the number of bytes from the beginning of one row to the beginning
// of the next.  The format is the pixel format of the buffer and
// must be one of those advertised through the wl_shm.format event.
// 
// A buffer will keep a reference to the pool it was created from
// so it is valid to destroy the pool immediately after creating
// a buffer from it.
func (this *ShmPool) CreateBuffer(offset int32, width int32, height int32, stride int32, format uint32) (*Buffer, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewBuffer()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())
	binary.Write(this.c.buf, hostByteOrder, offset)
	binary.Write(this.c.buf, hostByteOrder, width)
	binary.Write(this.c.buf, hostByteOrder, height)
	binary.Write(this.c.buf, hostByteOrder, stride)
	binary.Write(this.c.buf, hostByteOrder, format)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShmPoolCreateBuffer)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}
// Destroy the shared memory pool.
// 
// The mmapped memory will be released when all
// buffers that have been created from this pool
// are gone.
func (this *ShmPool) Destroy() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShmPoolDestroy)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This request will cause the server to remap the backing memory
// for the pool from the file descriptor passed when the pool was
// created, but using the new size.  This request can only be
// used to make the pool bigger.
func (this *ShmPool) Resize(size int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, size)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShmPoolResize)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    ShmErrorInvalidFormat = 0 // buffer format is not known
    ShmErrorInvalidStride = 1 // invalid size or stride during pool or buffer creation
    ShmErrorInvalidFd = 2 // mmapping the file descriptor failed
)

const (
    ShmFormatArgb8888 = 0 // 32-bit ARGB format, [31:0] A:R:G:B 8:8:8:8 little endian
    ShmFormatXrgb8888 = 1 // 32-bit RGB format, [31:0] x:R:G:B 8:8:8:8 little endian
    ShmFormatC8 = 0x20203843 // 8-bit color index format, [7:0] C
    ShmFormatRgb332 = 0x38424752 // 8-bit RGB format, [7:0] R:G:B 3:3:2
    ShmFormatBgr233 = 0x38524742 // 8-bit BGR format, [7:0] B:G:R 2:3:3
    ShmFormatXrgb4444 = 0x32315258 // 16-bit xRGB format, [15:0] x:R:G:B 4:4:4:4 little endian
    ShmFormatXbgr4444 = 0x32314258 // 16-bit xBGR format, [15:0] x:B:G:R 4:4:4:4 little endian
    ShmFormatRgbx4444 = 0x32315852 // 16-bit RGBx format, [15:0] R:G:B:x 4:4:4:4 little endian
    ShmFormatBgrx4444 = 0x32315842 // 16-bit BGRx format, [15:0] B:G:R:x 4:4:4:4 little endian
    ShmFormatArgb4444 = 0x32315241 // 16-bit ARGB format, [15:0] A:R:G:B 4:4:4:4 little endian
    ShmFormatAbgr4444 = 0x32314241 // 16-bit ABGR format, [15:0] A:B:G:R 4:4:4:4 little endian
    ShmFormatRgba4444 = 0x32314152 // 16-bit RBGA format, [15:0] R:G:B:A 4:4:4:4 little endian
    ShmFormatBgra4444 = 0x32314142 // 16-bit BGRA format, [15:0] B:G:R:A 4:4:4:4 little endian
    ShmFormatXrgb1555 = 0x35315258 // 16-bit xRGB format, [15:0] x:R:G:B 1:5:5:5 little endian
    ShmFormatXbgr1555 = 0x35314258 // 16-bit xBGR 1555 format, [15:0] x:B:G:R 1:5:5:5 little endian
    ShmFormatRgbx5551 = 0x35315852 // 16-bit RGBx 5551 format, [15:0] R:G:B:x 5:5:5:1 little endian
    ShmFormatBgrx5551 = 0x35315842 // 16-bit BGRx 5551 format, [15:0] B:G:R:x 5:5:5:1 little endian
    ShmFormatArgb1555 = 0x35315241 // 16-bit ARGB 1555 format, [15:0] A:R:G:B 1:5:5:5 little endian
    ShmFormatAbgr1555 = 0x35314241 // 16-bit ABGR 1555 format, [15:0] A:B:G:R 1:5:5:5 little endian
    ShmFormatRgba5551 = 0x35314152 // 16-bit RGBA 5551 format, [15:0] R:G:B:A 5:5:5:1 little endian
    ShmFormatBgra5551 = 0x35314142 // 16-bit BGRA 5551 format, [15:0] B:G:R:A 5:5:5:1 little endian
    ShmFormatRgb565 = 0x36314752 // 16-bit RGB 565 format, [15:0] R:G:B 5:6:5 little endian
    ShmFormatBgr565 = 0x36314742 // 16-bit BGR 565 format, [15:0] B:G:R 5:6:5 little endian
    ShmFormatRgb888 = 0x34324752 // 24-bit RGB format, [23:0] R:G:B little endian
    ShmFormatBgr888 = 0x34324742 // 24-bit BGR format, [23:0] B:G:R little endian
    ShmFormatXbgr8888 = 0x34324258 // 32-bit xBGR format, [31:0] x:B:G:R 8:8:8:8 little endian
    ShmFormatRgbx8888 = 0x34325852 // 32-bit RGBx format, [31:0] R:G:B:x 8:8:8:8 little endian
    ShmFormatBgrx8888 = 0x34325842 // 32-bit BGRx format, [31:0] B:G:R:x 8:8:8:8 little endian
    ShmFormatAbgr8888 = 0x34324241 // 32-bit ABGR format, [31:0] A:B:G:R 8:8:8:8 little endian
    ShmFormatRgba8888 = 0x34324152 // 32-bit RGBA format, [31:0] R:G:B:A 8:8:8:8 little endian
    ShmFormatBgra8888 = 0x34324142 // 32-bit BGRA format, [31:0] B:G:R:A 8:8:8:8 little endian
    ShmFormatXrgb2101010 = 0x30335258 // 32-bit xRGB format, [31:0] x:R:G:B 2:10:10:10 little endian
    ShmFormatXbgr2101010 = 0x30334258 // 32-bit xBGR format, [31:0] x:B:G:R 2:10:10:10 little endian
    ShmFormatRgbx1010102 = 0x30335852 // 32-bit RGBx format, [31:0] R:G:B:x 10:10:10:2 little endian
    ShmFormatBgrx1010102 = 0x30335842 // 32-bit BGRx format, [31:0] B:G:R:x 10:10:10:2 little endian
    ShmFormatArgb2101010 = 0x30335241 // 32-bit ARGB format, [31:0] A:R:G:B 2:10:10:10 little endian
    ShmFormatAbgr2101010 = 0x30334241 // 32-bit ABGR format, [31:0] A:B:G:R 2:10:10:10 little endian
    ShmFormatRgba1010102 = 0x30334152 // 32-bit RGBA format, [31:0] R:G:B:A 10:10:10:2 little endian
    ShmFormatBgra1010102 = 0x30334142 // 32-bit BGRA format, [31:0] B:G:R:A 10:10:10:2 little endian
    ShmFormatYuyv = 0x56595559 // packed YCbCr format, [31:0] Cr0:Y1:Cb0:Y0 8:8:8:8 little endian
    ShmFormatYvyu = 0x55595659 // packed YCbCr format, [31:0] Cb0:Y1:Cr0:Y0 8:8:8:8 little endian
    ShmFormatUyvy = 0x59565955 // packed YCbCr format, [31:0] Y1:Cr0:Y0:Cb0 8:8:8:8 little endian
    ShmFormatVyuy = 0x59555956 // packed YCbCr format, [31:0] Y1:Cb0:Y0:Cr0 8:8:8:8 little endian
    ShmFormatAyuv = 0x56555941 // packed AYCbCr format, [31:0] A:Y:Cb:Cr 8:8:8:8 little endian
    ShmFormatNv12 = 0x3231564e // 2 plane YCbCr Cr:Cb format, 2x2 subsampled Cr:Cb plane
    ShmFormatNv21 = 0x3132564e // 2 plane YCbCr Cb:Cr format, 2x2 subsampled Cb:Cr plane
    ShmFormatNv16 = 0x3631564e // 2 plane YCbCr Cr:Cb format, 2x1 subsampled Cr:Cb plane
    ShmFormatNv61 = 0x3136564e // 2 plane YCbCr Cb:Cr format, 2x1 subsampled Cb:Cr plane
    ShmFormatYuv410 = 0x39565559 // 3 plane YCbCr format, 4x4 subsampled Cb (1) and Cr (2) planes
    ShmFormatYvu410 = 0x39555659 // 3 plane YCbCr format, 4x4 subsampled Cr (1) and Cb (2) planes
    ShmFormatYuv411 = 0x31315559 // 3 plane YCbCr format, 4x1 subsampled Cb (1) and Cr (2) planes
    ShmFormatYvu411 = 0x31315659 // 3 plane YCbCr format, 4x1 subsampled Cr (1) and Cb (2) planes
    ShmFormatYuv420 = 0x32315559 // 3 plane YCbCr format, 2x2 subsampled Cb (1) and Cr (2) planes
    ShmFormatYvu420 = 0x32315659 // 3 plane YCbCr format, 2x2 subsampled Cr (1) and Cb (2) planes
    ShmFormatYuv422 = 0x36315559 // 3 plane YCbCr format, 2x1 subsampled Cb (1) and Cr (2) planes
    ShmFormatYvu422 = 0x36315659 // 3 plane YCbCr format, 2x1 subsampled Cr (1) and Cb (2) planes
    ShmFormatYuv444 = 0x34325559 // 3 plane YCbCr format, non-subsampled Cb (1) and Cr (2) planes
    ShmFormatYvu444 = 0x34325659 // 3 plane YCbCr format, non-subsampled Cr (1) and Cb (2) planes
)

const (
    opCodeShmFormat = 0
)

const (
	opCodeShmCreatePool = 0
)

// Shm Events
//
// Format
// Informs the client about a valid pixel format that
// can be used for buffers. Known formats include
// argb8888 and xrgb8888.
type ShmListener interface {
    Format(format uint32)
}

// A singleton global object that provides support for shared
// memory.
// 
// Clients can create wl_shm_pool objects using the create_pool
// request.
// 
// At connection setup time, the wl_shm object emits one or more
// format events to inform clients about the valid pixel formats
// that can be used for buffers.
type Shm struct {
    id ObjectID
    l  ShmListener
	c  *Client
}

func (c *Client) NewShm() *Shm {
	o := &Shm{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Shm) ID() ObjectID {
	return this.id
}

func (this *Shm) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeShmFormat:
		if this.l == nil {
			fmt.Println("ignoring Format event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			format := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Format(format)
        }
	
    }
}

func (this *Shm) SetListener(listener ShmListener) {
    this.l = listener
}

// Create a new wl_shm_pool object.
// 
// The pool can be used to create shared memory based buffer
// objects.  The server will mmap size bytes of the passed file
// descriptor, to use as backing memory for the pool.
func (this *Shm) CreatePool(fd *os.File, size int32) (*ShmPool, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewShmPool()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())
	// TODO: handle fds
	binary.Write(this.c.buf, hostByteOrder, size)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShmCreatePool)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}

const (
    opCodeBufferRelease = 0
)

const (
	opCodeBufferDestroy = 0
)

// Buffer Events
//
// Release
// Sent when this wl_buffer is no longer used by the compositor.
// The client is now free to reuse or destroy this buffer and its
// backing storage.
// 
// If a client receives a release event before the frame callback
// requested in the same wl_surface.commit that attaches this
// wl_buffer to a surface, then the client is immediately free to
// reuse the buffer and its backing storage, and does not need a
// second buffer for the next surface content update. Typically
// this is possible, when the compositor maintains a copy of the
// wl_surface contents, e.g. as a GL texture. This is an important
// optimization for GL(ES) compositors with wl_shm clients.
type BufferListener interface {
    Release()
}

// A buffer provides the content for a wl_surface. Buffers are
// created through factory interfaces such as wl_drm, wl_shm or
// similar. It has a width and a height and can be attached to a
// wl_surface, but the mechanism by which a client provides and
// updates the contents is defined by the buffer factory interface.
type Buffer struct {
    id ObjectID
    l  BufferListener
	c  *Client
}

func (c *Client) NewBuffer() *Buffer {
	o := &Buffer{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Buffer) ID() ObjectID {
	return this.id
}

func (this *Buffer) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeBufferRelease:
		if this.l == nil {
			fmt.Println("ignoring Release event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.Release()
        }
	
    }
}

func (this *Buffer) SetListener(listener BufferListener) {
    this.l = listener
}

// Destroy a buffer. If and how you need to release the backing
// storage is defined by the buffer factory interface.
// 
// For possible side-effects to a surface, see wl_surface.attach.
func (this *Buffer) Destroy() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeBufferDestroy)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    DataOfferErrorInvalidFinish = 0 // finish request was called untimely
    DataOfferErrorInvalidActionMask = 1 // action mask contains invalid values
    DataOfferErrorInvalidAction = 2 // action argument has an invalid value
    DataOfferErrorInvalidOffer = 3 // offer doesn't accept this request
)

const (
    opCodeDataOfferOffer = 0
    opCodeDataOfferSourceActions = 1
    opCodeDataOfferAction = 2
)

const (
	opCodeDataOfferAccept = 0
	opCodeDataOfferReceive = 1
	opCodeDataOfferDestroy = 2
	opCodeDataOfferFinish = 3
	opCodeDataOfferSetActions = 4
)

// DataOffer Events
//
// Offer
// Sent immediately after creating the wl_data_offer object.  One
// event per offered mime type.
//
// SourceActions
// This event indicates the actions offered by the data source. It
// will be sent right after wl_data_device.enter, or anytime the source
// side changes its offered actions through wl_data_source.set_actions.
//
// Action
// This event indicates the action selected by the compositor after
// matching the source/destination side actions. Only one action (or
// none) will be offered here.
// 
// This event can be emitted multiple times during the drag-and-drop
// operation in response to destination side action changes through
// wl_data_offer.set_actions.
// 
// This event will no longer be emitted after wl_data_device.drop
// happened on the drag-and-drop destination, the client must
// honor the last action received, or the last preferred one set
// through wl_data_offer.set_actions when handling an "ask" action.
// 
// Compositors may also change the selected action on the fly, mainly
// in response to keyboard modifier changes during the drag-and-drop
// operation.
// 
// The most recent action received is always the valid one. Prior to
// receiving wl_data_device.drop, the chosen action may change (e.g.
// due to keyboard modifiers being pressed). At the time of receiving
// wl_data_device.drop the drag-and-drop destination must honor the
// last action received.
// 
// Action changes may still happen after wl_data_device.drop,
// especially on "ask" actions, where the drag-and-drop destination
// may choose another action afterwards. Action changes happening
// at this stage are always the result of inter-client negotiation, the
// compositor shall no longer be able to induce a different action.
// 
// Upon "ask" actions, it is expected that the drag-and-drop destination
// may potentially choose a different action and/or mime type,
// based on wl_data_offer.source_actions and finally chosen by the
// user (e.g. popping up a menu with the available options). The
// final wl_data_offer.set_actions and wl_data_offer.accept requests
// must happen before the call to wl_data_offer.finish.
type DataOfferListener interface {
    Offer(mimeType string)
    SourceActions(sourceActions uint32)
    Action(dndAction uint32)
}

// A wl_data_offer represents a piece of data offered for transfer
// by another client (the source client).  It is used by the
// copy-and-paste and drag-and-drop mechanisms.  The offer
// describes the different mime types that the data can be
// converted to and provides the mechanism for transferring the
// data directly from the source client.
type DataOffer struct {
    id ObjectID
    l  DataOfferListener
	c  *Client
}

func (c *Client) NewDataOffer() *DataOffer {
	o := &DataOffer{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *DataOffer) ID() ObjectID {
	return this.id
}

func (this *DataOffer) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeDataOfferOffer:
		if this.l == nil {
			fmt.Println("ignoring Offer event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			len = int(hostByteOrder.Uint32(payload[0+off:4+off]))
			mimeType := string(payload[4+off:4+len+off-1])
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}

			this.l.Offer(mimeType)
        }
	case opCodeDataOfferSourceActions:
		if this.l == nil {
			fmt.Println("ignoring SourceActions event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			sourceActions := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.SourceActions(sourceActions)
        }
	case opCodeDataOfferAction:
		if this.l == nil {
			fmt.Println("ignoring Action event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			dndAction := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Action(dndAction)
        }
	
    }
}

func (this *DataOffer) SetListener(listener DataOfferListener) {
    this.l = listener
}

// Indicate that the client can accept the given mime type, or
// NULL for not accepted.
// 
// For objects of version 2 or older, this request is used by the
// client to give feedback whether the client can receive the given
// mime type, or NULL if none is accepted; the feedback does not
// determine whether the drag-and-drop operation succeeds or not.
// 
// For objects of version 3 or newer, this request determines the
// final result of the drag-and-drop operation. If the end result
// is that no mime types were accepted, the drag-and-drop operation
// will be cancelled and the corresponding drag source will receive
// wl_data_source.cancelled. Clients may still use this event in
// conjunction with wl_data_source.action for feedback.
func (this *DataOffer) Accept(serial uint32, mimeType string) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, serial)
	binary.Write(this.c.buf, hostByteOrder, len(mimeType))
	this.c.buf.WriteString(mimeType)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataOfferAccept)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// To transfer the offered data, the client issues this request
// and indicates the mime type it wants to receive.  The transfer
// happens through the passed file descriptor (typically created
// with the pipe system call).  The source client writes the data
// in the mime type representation requested and then closes the
// file descriptor.
// 
// The receiving client reads from the read end of the pipe until
// EOF and then closes its end, at which point the transfer is
// complete.
// 
// This request may happen multiple times for different mime types,
// both before and after wl_data_device.drop. Drag-and-drop destination
// clients may preemptively fetch data or examine it more closely to
// determine acceptance.
func (this *DataOffer) Receive(mimeType string, fd *os.File) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, len(mimeType))
	this.c.buf.WriteString(mimeType)
	// TODO: handle fds

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataOfferReceive)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Destroy the data offer.
func (this *DataOffer) Destroy() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataOfferDestroy)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Notifies the compositor that the drag destination successfully
// finished the drag-and-drop operation.
// 
// Upon receiving this request, the compositor will emit
// wl_data_source.dnd_finished on the drag source client.
// 
// It is a client error to perform other requests than
// wl_data_offer.destroy after this one. It is also an error to perform
// this request after a NULL mime type has been set in
// wl_data_offer.accept or no action was received through
// wl_data_offer.action.
func (this *DataOffer) Finish() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataOfferFinish)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Sets the actions that the destination side client supports for
// this operation. This request may trigger the emission of
// wl_data_source.action and wl_data_offer.action events if the compositor
// needs to change the selected action.
// 
// This request can be called multiple times throughout the
// drag-and-drop operation, typically in response to wl_data_device.enter
// or wl_data_device.motion events.
// 
// This request determines the final result of the drag-and-drop
// operation. If the end result is that no action is accepted,
// the drag source will receive wl_drag_source.cancelled.
// 
// The dnd_actions argument must contain only values expressed in the
// wl_data_device_manager.dnd_actions enum, and the preferred_action
// argument must only contain one of those values set, otherwise it
// will result in a protocol error.
// 
// While managing an "ask" action, the destination drag-and-drop client
// may perform further wl_data_offer.receive requests, and is expected
// to perform one last wl_data_offer.set_actions request with a preferred
// action other than "ask" (and optionally wl_data_offer.accept) before
// requesting wl_data_offer.finish, in order to convey the action selected
// by the user. If the preferred action is not in the
// wl_data_offer.source_actions mask, an error will be raised.
// 
// If the "ask" action is dismissed (e.g. user cancellation), the client
// is expected to perform wl_data_offer.destroy right away.
// 
// This request can only be made on drag-and-drop offers, a protocol error
// will be raised otherwise.
func (this *DataOffer) SetActions(dndActions uint32, preferredAction uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, dndActions)
	binary.Write(this.c.buf, hostByteOrder, preferredAction)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataOfferSetActions)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    DataSourceErrorInvalidActionMask = 0 // action mask contains invalid values
    DataSourceErrorInvalidSource = 1 // source doesn't accept this request
)

const (
    opCodeDataSourceTarget = 0
    opCodeDataSourceSend = 1
    opCodeDataSourceCancelled = 2
    opCodeDataSourceDndDropPerformed = 3
    opCodeDataSourceDndFinished = 4
    opCodeDataSourceAction = 5
)

const (
	opCodeDataSourceOffer = 0
	opCodeDataSourceDestroy = 1
	opCodeDataSourceSetActions = 2
)

// DataSource Events
//
// Target
// Sent when a target accepts pointer_focus or motion events.  If
// a target does not accept any of the offered types, type is NULL.
// 
// Used for feedback during drag-and-drop.
//
// Send
// Request for data from the client.  Send the data as the
// specified mime type over the passed file descriptor, then
// close it.
//
// Cancelled
// This data source is no longer valid. There are several reasons why
// this could happen:
// 
// - The data source has been replaced by another data source.
// - The drag-and-drop operation was performed, but the drop destination
// did not accept any of the mime types offered through
// wl_data_source.target.
// - The drag-and-drop operation was performed, but the drop destination
// did not select any of the actions present in the mask offered through
// wl_data_source.action.
// - The drag-and-drop operation was performed but didn't happen over a
// surface.
// - The compositor cancelled the drag-and-drop operation (e.g. compositor
// dependent timeouts to avoid stale drag-and-drop transfers).
// 
// The client should clean up and destroy this data source.
// 
// For objects of version 2 or older, wl_data_source.cancelled will
// only be emitted if the data source was replaced by another data
// source.
//
// DndDropPerformed
// The user performed the drop action. This event does not indicate
// acceptance, wl_data_source.cancelled may still be emitted afterwards
// if the drop destination does not accept any mime type.
// 
// However, this event might however not be received if the compositor
// cancelled the drag-and-drop operation before this event could happen.
// 
// Note that the data_source may still be used in the future and should
// not be destroyed here.
//
// DndFinished
// The drop destination finished interoperating with this data
// source, so the client is now free to destroy this data source and
// free all associated data.
// 
// If the action used to perform the operation was "move", the
// source can now delete the transferred data.
//
// Action
// This event indicates the action selected by the compositor after
// matching the source/destination side actions. Only one action (or
// none) will be offered here.
// 
// This event can be emitted multiple times during the drag-and-drop
// operation, mainly in response to destination side changes through
// wl_data_offer.set_actions, and as the data device enters/leaves
// surfaces.
// 
// It is only possible to receive this event after
// wl_data_source.dnd_drop_performed if the drag-and-drop operation
// ended in an "ask" action, in which case the final wl_data_source.action
// event will happen immediately before wl_data_source.dnd_finished.
// 
// Compositors may also change the selected action on the fly, mainly
// in response to keyboard modifier changes during the drag-and-drop
// operation.
// 
// The most recent action received is always the valid one. The chosen
// action may change alongside negotiation (e.g. an "ask" action can turn
// into a "move" operation), so the effects of the final action must
// always be applied in wl_data_offer.dnd_finished.
// 
// Clients can trigger cursor surface changes from this point, so
// they reflect the current action.
type DataSourceListener interface {
    Target(mimeType string)
    Send(mimeType string, fd *os.File)
    Cancelled()
    DndDropPerformed()
    DndFinished()
    Action(dndAction uint32)
}

// The wl_data_source object is the source side of a wl_data_offer.
// It is created by the source client in a data transfer and
// provides a way to describe the offered data and a way to respond
// to requests to transfer the data.
type DataSource struct {
    id ObjectID
    l  DataSourceListener
	c  *Client
}

func (c *Client) NewDataSource() *DataSource {
	o := &DataSource{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *DataSource) ID() ObjectID {
	return this.id
}

func (this *DataSource) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeDataSourceTarget:
		if this.l == nil {
			fmt.Println("ignoring Target event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			len = int(hostByteOrder.Uint32(payload[0+off:4+off]))
			mimeType := string(payload[4+off:4+len+off-1])
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}

			this.l.Target(mimeType)
        }
	case opCodeDataSourceSend:
		if this.l == nil {
			fmt.Println("ignoring Send event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			len = int(hostByteOrder.Uint32(payload[0+off:4+off]))
			mimeType := string(payload[4+off:4+len+off-1])
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}
			fd := file

			this.l.Send(mimeType, fd)
        }
	case opCodeDataSourceCancelled:
		if this.l == nil {
			fmt.Println("ignoring Cancelled event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.Cancelled()
        }
	case opCodeDataSourceDndDropPerformed:
		if this.l == nil {
			fmt.Println("ignoring DndDropPerformed event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.DndDropPerformed()
        }
	case opCodeDataSourceDndFinished:
		if this.l == nil {
			fmt.Println("ignoring DndFinished event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.DndFinished()
        }
	case opCodeDataSourceAction:
		if this.l == nil {
			fmt.Println("ignoring Action event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			dndAction := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Action(dndAction)
        }
	
    }
}

func (this *DataSource) SetListener(listener DataSourceListener) {
    this.l = listener
}

// This request adds a mime type to the set of mime types
// advertised to targets.  Can be called several times to offer
// multiple types.
func (this *DataSource) Offer(mimeType string) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, len(mimeType))
	this.c.buf.WriteString(mimeType)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataSourceOffer)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Destroy the data source.
func (this *DataSource) Destroy() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataSourceDestroy)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Sets the actions that the source side client supports for this
// operation. This request may trigger wl_data_source.action and
// wl_data_offer.action events if the compositor needs to change the
// selected action.
// 
// The dnd_actions argument must contain only values expressed in the
// wl_data_device_manager.dnd_actions enum, otherwise it will result
// in a protocol error.
// 
// This request must be made once only, and can only be made on sources
// used in drag-and-drop, so it must be performed before
// wl_data_device.start_drag. Attempting to use the source other than
// for drag-and-drop will raise a protocol error.
func (this *DataSource) SetActions(dndActions uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, dndActions)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataSourceSetActions)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    DataDeviceErrorRole = 0 // given wl_surface has another role
)

const (
    opCodeDataDeviceDataOffer = 0
    opCodeDataDeviceEnter = 1
    opCodeDataDeviceLeave = 2
    opCodeDataDeviceMotion = 3
    opCodeDataDeviceDrop = 4
    opCodeDataDeviceSelection = 5
)

const (
	opCodeDataDeviceStartDrag = 0
	opCodeDataDeviceSetSelection = 1
	opCodeDataDeviceRelease = 2
)

// DataDevice Events
//
// DataOffer
// The data_offer event introduces a new wl_data_offer object,
// which will subsequently be used in either the
// data_device.enter event (for drag-and-drop) or the
// data_device.selection event (for selections).  Immediately
// following the data_device_data_offer event, the new data_offer
// object will send out data_offer.offer events to describe the
// mime types it offers.
//
// Enter
// This event is sent when an active drag-and-drop pointer enters
// a surface owned by the client.  The position of the pointer at
// enter time is provided by the x and y arguments, in surface-local
// coordinates.
//
// Leave
// This event is sent when the drag-and-drop pointer leaves the
// surface and the session ends.  The client must destroy the
// wl_data_offer introduced at enter time at this point.
//
// Motion
// This event is sent when the drag-and-drop pointer moves within
// the currently focused surface. The new position of the pointer
// is provided by the x and y arguments, in surface-local
// coordinates.
//
// Drop
// The event is sent when a drag-and-drop operation is ended
// because the implicit grab is removed.
// 
// The drag-and-drop destination is expected to honor the last action
// received through wl_data_offer.action, if the resulting action is
// "copy" or "move", the destination can still perform
// wl_data_offer.receive requests, and is expected to end all
// transfers with a wl_data_offer.finish request.
// 
// If the resulting action is "ask", the action will not be considered
// final. The drag-and-drop destination is expected to perform one last
// wl_data_offer.set_actions request, or wl_data_offer.destroy in order
// to cancel the operation.
//
// Selection
// The selection event is sent out to notify the client of a new
// wl_data_offer for the selection for this device.  The
// data_device.data_offer and the data_offer.offer events are
// sent out immediately before this event to introduce the data
// offer object.  The selection event is sent to a client
// immediately before receiving keyboard focus and when a new
// selection is set while the client has keyboard focus.  The
// data_offer is valid until a new data_offer or NULL is received
// or until the client loses keyboard focus.  The client must
// destroy the previous selection data_offer, if any, upon receiving
// this event.
type DataDeviceListener interface {
    DataOffer()
    Enter(serial uint32, surface uint32, x float64, y float64, id uint32)
    Leave()
    Motion(time uint32, x float64, y float64)
    Drop()
    Selection(id uint32)
}

// There is one wl_data_device per seat which can be obtained
// from the global wl_data_device_manager singleton.
// 
// A wl_data_device provides access to inter-client data transfer
// mechanisms such as copy-and-paste and drag-and-drop.
type DataDevice struct {
    id ObjectID
    l  DataDeviceListener
	c  *Client
}

func (c *Client) NewDataDevice() *DataDevice {
	o := &DataDevice{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *DataDevice) ID() ObjectID {
	return this.id
}

func (this *DataDevice) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeDataDeviceDataOffer:
		if this.l == nil {
			fmt.Println("ignoring DataOffer event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.DataOffer()
        }
	case opCodeDataDeviceEnter:
		if this.l == nil {
			fmt.Println("ignoring Enter event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			surface := hostByteOrder.Uint32(payload[4+off:8+off])
			x := fixedToFloat64(int32(hostByteOrder.Uint32(payload[8+off:12+off])))
			y := fixedToFloat64(int32(hostByteOrder.Uint32(payload[12+off:16+off])))
			id := hostByteOrder.Uint32(payload[16+off:20+off])

			this.l.Enter(serial, surface, x, y, id)
        }
	case opCodeDataDeviceLeave:
		if this.l == nil {
			fmt.Println("ignoring Leave event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.Leave()
        }
	case opCodeDataDeviceMotion:
		if this.l == nil {
			fmt.Println("ignoring Motion event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			time := hostByteOrder.Uint32(payload[0+off:4+off])
			x := fixedToFloat64(int32(hostByteOrder.Uint32(payload[4+off:8+off])))
			y := fixedToFloat64(int32(hostByteOrder.Uint32(payload[8+off:12+off])))

			this.l.Motion(time, x, y)
        }
	case opCodeDataDeviceDrop:
		if this.l == nil {
			fmt.Println("ignoring Drop event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.Drop()
        }
	case opCodeDataDeviceSelection:
		if this.l == nil {
			fmt.Println("ignoring Selection event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			id := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Selection(id)
        }
	
    }
}

func (this *DataDevice) SetListener(listener DataDeviceListener) {
    this.l = listener
}

// This request asks the compositor to start a drag-and-drop
// operation on behalf of the client.
// 
// The source argument is the data source that provides the data
// for the eventual data transfer. If source is NULL, enter, leave
// and motion events are sent only to the client that initiated the
// drag and the client is expected to handle the data passing
// internally.
// 
// The origin surface is the surface where the drag originates and
// the client must have an active implicit grab that matches the
// serial.
// 
// The icon surface is an optional (can be NULL) surface that
// provides an icon to be moved around with the cursor.  Initially,
// the top-left corner of the icon surface is placed at the cursor
// hotspot, but subsequent wl_surface.attach request can move the
// relative position. Attach requests must be confirmed with
// wl_surface.commit as usual. The icon surface is given the role of
// a drag-and-drop icon. If the icon surface already has another role,
// it raises a protocol error.
// 
// The current and pending input regions of the icon wl_surface are
// cleared, and wl_surface.set_input_region is ignored until the
// wl_surface is no longer used as the icon surface. When the use
// as an icon ends, the current and pending input regions become
// undefined, and the wl_surface is unmapped.
func (this *DataDevice) StartDrag(source uint32, origin uint32, icon uint32, serial uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, source)
	binary.Write(this.c.buf, hostByteOrder, origin)
	binary.Write(this.c.buf, hostByteOrder, icon)
	binary.Write(this.c.buf, hostByteOrder, serial)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataDeviceStartDrag)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This request asks the compositor to set the selection
// to the data from the source on behalf of the client.
// 
// To unset the selection, set the source to NULL.
func (this *DataDevice) SetSelection(source uint32, serial uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, source)
	binary.Write(this.c.buf, hostByteOrder, serial)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataDeviceSetSelection)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This request destroys the data device.
func (this *DataDevice) Release() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataDeviceRelease)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    DataDeviceManagerDndActionNone = 0 // no action
    DataDeviceManagerDndActionCopy = 1 // copy action
    DataDeviceManagerDndActionMove = 2 // move action
    DataDeviceManagerDndActionAsk = 4 // ask action
)

const (
)

const (
	opCodeDataDeviceManagerCreateDataSource = 0
	opCodeDataDeviceManagerGetDataDevice = 1
)

// DataDeviceManager Events
type DataDeviceManagerListener interface {
}

// The wl_data_device_manager is a singleton global object that
// provides access to inter-client data transfer mechanisms such as
// copy-and-paste and drag-and-drop.  These mechanisms are tied to
// a wl_seat and this interface lets a client get a wl_data_device
// corresponding to a wl_seat.
// 
// Depending on the version bound, the objects created from the bound
// wl_data_device_manager object will have different requirements for
// functioning properly. See wl_data_source.set_actions,
// wl_data_offer.accept and wl_data_offer.finish for details.
type DataDeviceManager struct {
    id ObjectID
    l  DataDeviceManagerListener
	c  *Client
}

func (c *Client) NewDataDeviceManager() *DataDeviceManager {
	o := &DataDeviceManager{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *DataDeviceManager) ID() ObjectID {
	return this.id
}

func (this *DataDeviceManager) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	
    }
}

func (this *DataDeviceManager) SetListener(listener DataDeviceManagerListener) {
    this.l = listener
}

// Create a new data source.
func (this *DataDeviceManager) CreateDataSource() (*DataSource, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewDataSource()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataDeviceManagerCreateDataSource)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}
// Create a new data device for a given seat.
func (this *DataDeviceManager) GetDataDevice(seat uint32) (*DataDevice, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewDataDevice()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())
	binary.Write(this.c.buf, hostByteOrder, seat)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeDataDeviceManagerGetDataDevice)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}

const (
    ShellErrorRole = 0 // given wl_surface has another role
)

const (
)

const (
	opCodeShellGetShellSurface = 0
)

// Shell Events
type ShellListener interface {
}

// This interface is implemented by servers that provide
// desktop-style user interfaces.
// 
// It allows clients to associate a wl_shell_surface with
// a basic surface.
type Shell struct {
    id ObjectID
    l  ShellListener
	c  *Client
}

func (c *Client) NewShell() *Shell {
	o := &Shell{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Shell) ID() ObjectID {
	return this.id
}

func (this *Shell) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	
    }
}

func (this *Shell) SetListener(listener ShellListener) {
    this.l = listener
}

// Create a shell surface for an existing surface. This gives
// the wl_surface the role of a shell surface. If the wl_surface
// already has another role, it raises a protocol error.
// 
// Only one shell surface can be associated with a given surface.
func (this *Shell) GetShellSurface(surface uint32) (*ShellSurface, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewShellSurface()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())
	binary.Write(this.c.buf, hostByteOrder, surface)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellGetShellSurface)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}

const (
    ShellSurfaceResizeNone = 0 // no edge
    ShellSurfaceResizeTop = 1 // top edge
    ShellSurfaceResizeBottom = 2 // bottom edge
    ShellSurfaceResizeLeft = 4 // left edge
    ShellSurfaceResizeTopLeft = 5 // top and left edges
    ShellSurfaceResizeBottomLeft = 6 // bottom and left edges
    ShellSurfaceResizeRight = 8 // right edge
    ShellSurfaceResizeTopRight = 9 // top and right edges
    ShellSurfaceResizeBottomRight = 10 // bottom and right edges
)

const (
    ShellSurfaceTransientInactive = 0x1 // do not set keyboard focus
)

const (
    ShellSurfaceFullscreenMethodDefault = 0 // no preference, apply default policy
    ShellSurfaceFullscreenMethodScale = 1 // scale, preserve the surface's aspect ratio and center on output
    ShellSurfaceFullscreenMethodDriver = 2 // switch output mode to the smallest mode that can fit the surface, add black borders to compensate size mismatch
    ShellSurfaceFullscreenMethodFill = 3 // no upscaling, center on output and add black borders to compensate size mismatch
)

const (
    opCodeShellSurfacePing = 0
    opCodeShellSurfaceConfigure = 1
    opCodeShellSurfacePopupDone = 2
)

const (
	opCodeShellSurfacePong = 0
	opCodeShellSurfaceMove = 1
	opCodeShellSurfaceResize = 2
	opCodeShellSurfaceSetToplevel = 3
	opCodeShellSurfaceSetTransient = 4
	opCodeShellSurfaceSetFullscreen = 5
	opCodeShellSurfaceSetPopup = 6
	opCodeShellSurfaceSetMaximized = 7
	opCodeShellSurfaceSetTitle = 8
	opCodeShellSurfaceSetClass = 9
)

// ShellSurface Events
//
// Ping
// Ping a client to check if it is receiving events and sending
// requests. A client is expected to reply with a pong request.
//
// Configure
// The configure event asks the client to resize its surface.
// 
// The size is a hint, in the sense that the client is free to
// ignore it if it doesn't resize, pick a smaller size (to
// satisfy aspect ratio or resize in steps of NxM pixels).
// 
// The edges parameter provides a hint about how the surface
// was resized. The client may use this information to decide
// how to adjust its content to the new size (e.g. a scrolling
// area might adjust its content position to leave the viewable
// content unmoved).
// 
// The client is free to dismiss all but the last configure
// event it received.
// 
// The width and height arguments specify the size of the window
// in surface-local coordinates.
//
// PopupDone
// The popup_done event is sent out when a popup grab is broken,
// that is, when the user clicks a surface that doesn't belong
// to the client owning the popup surface.
type ShellSurfaceListener interface {
    Ping(serial uint32)
    Configure(edges uint32, width int32, height int32)
    PopupDone()
}

// An interface that may be implemented by a wl_surface, for
// implementations that provide a desktop-style user interface.
// 
// It provides requests to treat surfaces like toplevel, fullscreen
// or popup windows, move, resize or maximize them, associate
// metadata like title and class, etc.
// 
// On the server side the object is automatically destroyed when
// the related wl_surface is destroyed. On the client side,
// wl_shell_surface_destroy() must be called before destroying
// the wl_surface object.
type ShellSurface struct {
    id ObjectID
    l  ShellSurfaceListener
	c  *Client
}

func (c *Client) NewShellSurface() *ShellSurface {
	o := &ShellSurface{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *ShellSurface) ID() ObjectID {
	return this.id
}

func (this *ShellSurface) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeShellSurfacePing:
		if this.l == nil {
			fmt.Println("ignoring Ping event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Ping(serial)
        }
	case opCodeShellSurfaceConfigure:
		if this.l == nil {
			fmt.Println("ignoring Configure event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			edges := hostByteOrder.Uint32(payload[0+off:4+off])
			width := int32(hostByteOrder.Uint32(payload[4+off:8+off]))
			height := int32(hostByteOrder.Uint32(payload[8+off:12+off]))

			this.l.Configure(edges, width, height)
        }
	case opCodeShellSurfacePopupDone:
		if this.l == nil {
			fmt.Println("ignoring PopupDone event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.PopupDone()
        }
	
    }
}

func (this *ShellSurface) SetListener(listener ShellSurfaceListener) {
    this.l = listener
}

// A client must respond to a ping event with a pong request or
// the client may be deemed unresponsive.
func (this *ShellSurface) Pong(serial uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, serial)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfacePong)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Start a pointer-driven move of the surface.
// 
// This request must be used in response to a button press event.
// The server may ignore move requests depending on the state of
// the surface (e.g. fullscreen or maximized).
func (this *ShellSurface) Move(seat uint32, serial uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, seat)
	binary.Write(this.c.buf, hostByteOrder, serial)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceMove)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Start a pointer-driven resizing of the surface.
// 
// This request must be used in response to a button press event.
// The server may ignore resize requests depending on the state of
// the surface (e.g. fullscreen or maximized).
func (this *ShellSurface) Resize(seat uint32, serial uint32, edges uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, seat)
	binary.Write(this.c.buf, hostByteOrder, serial)
	binary.Write(this.c.buf, hostByteOrder, edges)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceResize)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Map the surface as a toplevel surface.
// 
// A toplevel surface is not fullscreen, maximized or transient.
func (this *ShellSurface) SetToplevel() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceSetToplevel)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Map the surface relative to an existing surface.
// 
// The x and y arguments specify the location of the upper left
// corner of the surface relative to the upper left corner of the
// parent surface, in surface-local coordinates.
// 
// The flags argument controls details of the transient behaviour.
func (this *ShellSurface) SetTransient(parent uint32, x int32, y int32, flags uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, parent)
	binary.Write(this.c.buf, hostByteOrder, x)
	binary.Write(this.c.buf, hostByteOrder, y)
	binary.Write(this.c.buf, hostByteOrder, flags)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceSetTransient)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Map the surface as a fullscreen surface.
// 
// If an output parameter is given then the surface will be made
// fullscreen on that output. If the client does not specify the
// output then the compositor will apply its policy - usually
// choosing the output on which the surface has the biggest surface
// area.
// 
// The client may specify a method to resolve a size conflict
// between the output size and the surface size - this is provided
// through the method parameter.
// 
// The framerate parameter is used only when the method is set
// to "driver", to indicate the preferred framerate. A value of 0
// indicates that the client does not care about framerate.  The
// framerate is specified in mHz, that is framerate of 60000 is 60Hz.
// 
// A method of "scale" or "driver" implies a scaling operation of
// the surface, either via a direct scaling operation or a change of
// the output mode. This will override any kind of output scaling, so
// that mapping a surface with a buffer size equal to the mode can
// fill the screen independent of buffer_scale.
// 
// A method of "fill" means we don't scale up the buffer, however
// any output scale is applied. This means that you may run into
// an edge case where the application maps a buffer with the same
// size of the output mode but buffer_scale 1 (thus making a
// surface larger than the output). In this case it is allowed to
// downscale the results to fit the screen.
// 
// The compositor must reply to this request with a configure event
// with the dimensions for the output on which the surface will
// be made fullscreen.
func (this *ShellSurface) SetFullscreen(method uint32, framerate uint32, output uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, method)
	binary.Write(this.c.buf, hostByteOrder, framerate)
	binary.Write(this.c.buf, hostByteOrder, output)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceSetFullscreen)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Map the surface as a popup.
// 
// A popup surface is a transient surface with an added pointer
// grab.
// 
// An existing implicit grab will be changed to owner-events mode,
// and the popup grab will continue after the implicit grab ends
// (i.e. releasing the mouse button does not cause the popup to
// be unmapped).
// 
// The popup grab continues until the window is destroyed or a
// mouse button is pressed in any other client's window. A click
// in any of the client's surfaces is reported as normal, however,
// clicks in other clients' surfaces will be discarded and trigger
// the callback.
// 
// The x and y arguments specify the location of the upper left
// corner of the surface relative to the upper left corner of the
// parent surface, in surface-local coordinates.
func (this *ShellSurface) SetPopup(seat uint32, serial uint32, parent uint32, x int32, y int32, flags uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, seat)
	binary.Write(this.c.buf, hostByteOrder, serial)
	binary.Write(this.c.buf, hostByteOrder, parent)
	binary.Write(this.c.buf, hostByteOrder, x)
	binary.Write(this.c.buf, hostByteOrder, y)
	binary.Write(this.c.buf, hostByteOrder, flags)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceSetPopup)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Map the surface as a maximized surface.
// 
// If an output parameter is given then the surface will be
// maximized on that output. If the client does not specify the
// output then the compositor will apply its policy - usually
// choosing the output on which the surface has the biggest surface
// area.
// 
// The compositor will reply with a configure event telling
// the expected new surface size. The operation is completed
// on the next buffer attach to this surface.
// 
// A maximized surface typically fills the entire output it is
// bound to, except for desktop elements such as panels. This is
// the main difference between a maximized shell surface and a
// fullscreen shell surface.
// 
// The details depend on the compositor implementation.
func (this *ShellSurface) SetMaximized(output uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, output)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceSetMaximized)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Set a short title for the surface.
// 
// This string may be used to identify the surface in a task bar,
// window list, or other user interface elements provided by the
// compositor.
// 
// The string must be encoded in UTF-8.
func (this *ShellSurface) SetTitle(title string) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, len(title))
	this.c.buf.WriteString(title)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceSetTitle)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Set a class for the surface.
// 
// The surface class identifies the general class of applications
// to which the surface belongs. A common convention is to use the
// file name (or the full path if it is a non-standard location) of
// the application's .desktop file as the class.
func (this *ShellSurface) SetClass(class string) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, len(class))
	this.c.buf.WriteString(class)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeShellSurfaceSetClass)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    SurfaceErrorInvalidScale = 0 // buffer scale value is invalid
    SurfaceErrorInvalidTransform = 1 // buffer transform value is invalid
)

const (
    opCodeSurfaceEnter = 0
    opCodeSurfaceLeave = 1
)

const (
	opCodeSurfaceDestroy = 0
	opCodeSurfaceAttach = 1
	opCodeSurfaceDamage = 2
	opCodeSurfaceFrame = 3
	opCodeSurfaceSetOpaqueRegion = 4
	opCodeSurfaceSetInputRegion = 5
	opCodeSurfaceCommit = 6
	opCodeSurfaceSetBufferTransform = 7
	opCodeSurfaceSetBufferScale = 8
	opCodeSurfaceDamageBuffer = 9
)

// Surface Events
//
// Enter
// This is emitted whenever a surface's creation, movement, or resizing
// results in some part of it being within the scanout region of an
// output.
// 
// Note that a surface may be overlapping with zero or more outputs.
//
// Leave
// This is emitted whenever a surface's creation, movement, or resizing
// results in it no longer having any part of it within the scanout region
// of an output.
type SurfaceListener interface {
    Enter(output uint32)
    Leave(output uint32)
}

// A surface is a rectangular area that is displayed on the screen.
// It has a location, size and pixel contents.
// 
// The size of a surface (and relative positions on it) is described
// in surface-local coordinates, which may differ from the buffer
// coordinates of the pixel content, in case a buffer_transform
// or a buffer_scale is used.
// 
// A surface without a "role" is fairly useless: a compositor does
// not know where, when or how to present it. The role is the
// purpose of a wl_surface. Examples of roles are a cursor for a
// pointer (as set by wl_pointer.set_cursor), a drag icon
// (wl_data_device.start_drag), a sub-surface
// (wl_subcompositor.get_subsurface), and a window as defined by a
// shell protocol (e.g. wl_shell.get_shell_surface).
// 
// A surface can have only one role at a time. Initially a
// wl_surface does not have a role. Once a wl_surface is given a
// role, it is set permanently for the whole lifetime of the
// wl_surface object. Giving the current role again is allowed,
// unless explicitly forbidden by the relevant interface
// specification.
// 
// Surface roles are given by requests in other interfaces such as
// wl_pointer.set_cursor. The request should explicitly mention
// that this request gives a role to a wl_surface. Often, this
// request also creates a new protocol object that represents the
// role and adds additional functionality to wl_surface. When a
// client wants to destroy a wl_surface, they must destroy this 'role
// object' before the wl_surface.
// 
// Destroying the role object does not remove the role from the
// wl_surface, but it may stop the wl_surface from "playing the role".
// For instance, if a wl_subsurface object is destroyed, the wl_surface
// it was created for will be unmapped and forget its position and
// z-order. It is allowed to create a wl_subsurface for the same
// wl_surface again, but it is not allowed to use the wl_surface as
// a cursor (cursor is a different role than sub-surface, and role
// switching is not allowed).
type Surface struct {
    id ObjectID
    l  SurfaceListener
	c  *Client
}

func (c *Client) NewSurface() *Surface {
	o := &Surface{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Surface) ID() ObjectID {
	return this.id
}

func (this *Surface) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeSurfaceEnter:
		if this.l == nil {
			fmt.Println("ignoring Enter event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			output := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Enter(output)
        }
	case opCodeSurfaceLeave:
		if this.l == nil {
			fmt.Println("ignoring Leave event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			output := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Leave(output)
        }
	
    }
}

func (this *Surface) SetListener(listener SurfaceListener) {
    this.l = listener
}

// Deletes the surface and invalidates its object ID.
func (this *Surface) Destroy() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceDestroy)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Set a buffer as the content of this surface.
// 
// The new size of the surface is calculated based on the buffer
// size transformed by the inverse buffer_transform and the
// inverse buffer_scale. This means that the supplied buffer
// must be an integer multiple of the buffer_scale.
// 
// The x and y arguments specify the location of the new pending
// buffer's upper left corner, relative to the current buffer's upper
// left corner, in surface-local coordinates. In other words, the
// x and y, combined with the new surface size define in which
// directions the surface's size changes.
// 
// Surface contents are double-buffered state, see wl_surface.commit.
// 
// The initial surface contents are void; there is no content.
// wl_surface.attach assigns the given wl_buffer as the pending
// wl_buffer. wl_surface.commit makes the pending wl_buffer the new
// surface contents, and the size of the surface becomes the size
// calculated from the wl_buffer, as described above. After commit,
// there is no pending buffer until the next attach.
// 
// Committing a pending wl_buffer allows the compositor to read the
// pixels in the wl_buffer. The compositor may access the pixels at
// any time after the wl_surface.commit request. When the compositor
// will not access the pixels anymore, it will send the
// wl_buffer.release event. Only after receiving wl_buffer.release,
// the client may reuse the wl_buffer. A wl_buffer that has been
// attached and then replaced by another attach instead of committed
// will not receive a release event, and is not used by the
// compositor.
// 
// Destroying the wl_buffer after wl_buffer.release does not change
// the surface contents. However, if the client destroys the
// wl_buffer before receiving the wl_buffer.release event, the surface
// contents become undefined immediately.
// 
// If wl_surface.attach is sent with a NULL wl_buffer, the
// following wl_surface.commit will remove the surface content.
func (this *Surface) Attach(buffer uint32, x int32, y int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, buffer)
	binary.Write(this.c.buf, hostByteOrder, x)
	binary.Write(this.c.buf, hostByteOrder, y)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceAttach)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This request is used to describe the regions where the pending
// buffer is different from the current surface contents, and where
// the surface therefore needs to be repainted. The compositor
// ignores the parts of the damage that fall outside of the surface.
// 
// Damage is double-buffered state, see wl_surface.commit.
// 
// The damage rectangle is specified in surface-local coordinates,
// where x and y specify the upper left corner of the damage rectangle.
// 
// The initial value for pending damage is empty: no damage.
// wl_surface.damage adds pending damage: the new pending damage
// is the union of old pending damage and the given rectangle.
// 
// wl_surface.commit assigns pending damage as the current damage,
// and clears pending damage. The server will clear the current
// damage as it repaints the surface.
// 
// Alternatively, damage can be posted with wl_surface.damage_buffer
// which uses buffer coordinates instead of surface coordinates,
// and is probably the preferred and intuitive way of doing this.
func (this *Surface) Damage(x int32, y int32, width int32, height int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, x)
	binary.Write(this.c.buf, hostByteOrder, y)
	binary.Write(this.c.buf, hostByteOrder, width)
	binary.Write(this.c.buf, hostByteOrder, height)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceDamage)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Request a notification when it is a good time to start drawing a new
// frame, by creating a frame callback. This is useful for throttling
// redrawing operations, and driving animations.
// 
// When a client is animating on a wl_surface, it can use the 'frame'
// request to get notified when it is a good time to draw and commit the
// next frame of animation. If the client commits an update earlier than
// that, it is likely that some updates will not make it to the display,
// and the client is wasting resources by drawing too often.
// 
// The frame request will take effect on the next wl_surface.commit.
// The notification will only be posted for one frame unless
// requested again. For a wl_surface, the notifications are posted in
// the order the frame requests were committed.
// 
// The server must send the notifications so that a client
// will not send excessive updates, while still allowing
// the highest possible update rate for clients that wait for the reply
// before drawing again. The server should give some time for the client
// to draw and commit after sending the frame callback events to let it
// hit the next output refresh.
// 
// A server should avoid signaling the frame callbacks if the
// surface is not visible in any way, e.g. the surface is off-screen,
// or completely obscured by other opaque surfaces.
// 
// The object returned by this request will be destroyed by the
// compositor after the callback is fired and as such the client must not
// attempt to use it after that point.
// 
// The callback_data passed in the callback is the current time, in
// milliseconds, with an undefined base.
func (this *Surface) Frame() (*Callback, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewCallback()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceFrame)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}
// This request sets the region of the surface that contains
// opaque content.
// 
// The opaque region is an optimization hint for the compositor
// that lets it optimize the redrawing of content behind opaque
// regions.  Setting an opaque region is not required for correct
// behaviour, but marking transparent content as opaque will result
// in repaint artifacts.
// 
// The opaque region is specified in surface-local coordinates.
// 
// The compositor ignores the parts of the opaque region that fall
// outside of the surface.
// 
// Opaque region is double-buffered state, see wl_surface.commit.
// 
// wl_surface.set_opaque_region changes the pending opaque region.
// wl_surface.commit copies the pending region to the current region.
// Otherwise, the pending and current regions are never changed.
// 
// The initial value for an opaque region is empty. Setting the pending
// opaque region has copy semantics, and the wl_region object can be
// destroyed immediately. A NULL wl_region causes the pending opaque
// region to be set to empty.
func (this *Surface) SetOpaqueRegion(region uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, region)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceSetOpaqueRegion)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This request sets the region of the surface that can receive
// pointer and touch events.
// 
// Input events happening outside of this region will try the next
// surface in the server surface stack. The compositor ignores the
// parts of the input region that fall outside of the surface.
// 
// The input region is specified in surface-local coordinates.
// 
// Input region is double-buffered state, see wl_surface.commit.
// 
// wl_surface.set_input_region changes the pending input region.
// wl_surface.commit copies the pending region to the current region.
// Otherwise the pending and current regions are never changed,
// except cursor and icon surfaces are special cases, see
// wl_pointer.set_cursor and wl_data_device.start_drag.
// 
// The initial value for an input region is infinite. That means the
// whole surface will accept input. Setting the pending input region
// has copy semantics, and the wl_region object can be destroyed
// immediately. A NULL wl_region causes the input region to be set
// to infinite.
func (this *Surface) SetInputRegion(region uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, region)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceSetInputRegion)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Surface state (input, opaque, and damage regions, attached buffers,
// etc.) is double-buffered. Protocol requests modify the pending state,
// as opposed to the current state in use by the compositor. A commit
// request atomically applies all pending state, replacing the current
// state. After commit, the new pending state is as documented for each
// related request.
// 
// On commit, a pending wl_buffer is applied first, and all other state
// second. This means that all coordinates in double-buffered state are
// relative to the new wl_buffer coming into use, except for
// wl_surface.attach itself. If there is no pending wl_buffer, the
// coordinates are relative to the current surface contents.
// 
// All requests that need a commit to become effective are documented
// to affect double-buffered state.
// 
// Other interfaces may add further double-buffered surface state.
func (this *Surface) Commit() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceCommit)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This request sets an optional transformation on how the compositor
// interprets the contents of the buffer attached to the surface. The
// accepted values for the transform parameter are the values for
// wl_output.transform.
// 
// Buffer transform is double-buffered state, see wl_surface.commit.
// 
// A newly created surface has its buffer transformation set to normal.
// 
// wl_surface.set_buffer_transform changes the pending buffer
// transformation. wl_surface.commit copies the pending buffer
// transformation to the current one. Otherwise, the pending and current
// values are never changed.
// 
// The purpose of this request is to allow clients to render content
// according to the output transform, thus permitting the compositor to
// use certain optimizations even if the display is rotated. Using
// hardware overlays and scanning out a client buffer for fullscreen
// surfaces are examples of such optimizations. Those optimizations are
// highly dependent on the compositor implementation, so the use of this
// request should be considered on a case-by-case basis.
// 
// Note that if the transform value includes 90 or 270 degree rotation,
// the width of the buffer will become the surface height and the height
// of the buffer will become the surface width.
// 
// If transform is not one of the values from the
// wl_output.transform enum the invalid_transform protocol error
// is raised.
func (this *Surface) SetBufferTransform(transform int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, transform)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceSetBufferTransform)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This request sets an optional scaling factor on how the compositor
// interprets the contents of the buffer attached to the window.
// 
// Buffer scale is double-buffered state, see wl_surface.commit.
// 
// A newly created surface has its buffer scale set to 1.
// 
// wl_surface.set_buffer_scale changes the pending buffer scale.
// wl_surface.commit copies the pending buffer scale to the current one.
// Otherwise, the pending and current values are never changed.
// 
// The purpose of this request is to allow clients to supply higher
// resolution buffer data for use on high resolution outputs. It is
// intended that you pick the same buffer scale as the scale of the
// output that the surface is displayed on. This means the compositor
// can avoid scaling when rendering the surface on that output.
// 
// Note that if the scale is larger than 1, then you have to attach
// a buffer that is larger (by a factor of scale in each dimension)
// than the desired surface size.
// 
// If scale is not positive the invalid_scale protocol error is
// raised.
func (this *Surface) SetBufferScale(scale int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, scale)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceSetBufferScale)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This request is used to describe the regions where the pending
// buffer is different from the current surface contents, and where
// the surface therefore needs to be repainted. The compositor
// ignores the parts of the damage that fall outside of the surface.
// 
// Damage is double-buffered state, see wl_surface.commit.
// 
// The damage rectangle is specified in buffer coordinates,
// where x and y specify the upper left corner of the damage rectangle.
// 
// The initial value for pending damage is empty: no damage.
// wl_surface.damage_buffer adds pending damage: the new pending
// damage is the union of old pending damage and the given rectangle.
// 
// wl_surface.commit assigns pending damage as the current damage,
// and clears pending damage. The server will clear the current
// damage as it repaints the surface.
// 
// This request differs from wl_surface.damage in only one way - it
// takes damage in buffer coordinates instead of surface-local
// coordinates. While this generally is more intuitive than surface
// coordinates, it is especially desirable when using wp_viewport
// or when a drawing library (like EGL) is unaware of buffer scale
// and buffer transform.
// 
// Note: Because buffer transformation changes and damage requests may
// be interleaved in the protocol stream, it is impossible to determine
// the actual mapping between surface and buffer damage until
// wl_surface.commit time. Therefore, compositors wishing to take both
// kinds of damage into account will have to accumulate damage from the
// two requests separately and only transform from one to the other
// after receiving the wl_surface.commit.
func (this *Surface) DamageBuffer(x int32, y int32, width int32, height int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, x)
	binary.Write(this.c.buf, hostByteOrder, y)
	binary.Write(this.c.buf, hostByteOrder, width)
	binary.Write(this.c.buf, hostByteOrder, height)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSurfaceDamageBuffer)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    SeatCapabilityPointer = 1 // the seat has pointer devices
    SeatCapabilityKeyboard = 2 // the seat has one or more keyboards
    SeatCapabilityTouch = 4 // the seat has touch devices
)

const (
    opCodeSeatCapabilities = 0
    opCodeSeatName = 1
)

const (
	opCodeSeatGetPointer = 0
	opCodeSeatGetKeyboard = 1
	opCodeSeatGetTouch = 2
	opCodeSeatRelease = 3
)

// Seat Events
//
// Capabilities
// This is emitted whenever a seat gains or loses the pointer,
// keyboard or touch capabilities.  The argument is a capability
// enum containing the complete set of capabilities this seat has.
// 
// When the pointer capability is added, a client may create a
// wl_pointer object using the wl_seat.get_pointer request. This object
// will receive pointer events until the capability is removed in the
// future.
// 
// When the pointer capability is removed, a client should destroy the
// wl_pointer objects associated with the seat where the capability was
// removed, using the wl_pointer.release request. No further pointer
// events will be received on these objects.
// 
// In some compositors, if a seat regains the pointer capability and a
// client has a previously obtained wl_pointer object of version 4 or
// less, that object may start sending pointer events again. This
// behavior is considered a misinterpretation of the intended behavior
// and must not be relied upon by the client. wl_pointer objects of
// version 5 or later must not send events if created before the most
// recent event notifying the client of an added pointer capability.
// 
// The above behavior also applies to wl_keyboard and wl_touch with the
// keyboard and touch capabilities, respectively.
//
// Name
// In a multiseat configuration this can be used by the client to help
// identify which physical devices the seat represents. Based on
// the seat configuration used by the compositor.
type SeatListener interface {
    Capabilities(capabilities uint32)
    Name(name string)
}

// A seat is a group of keyboards, pointer and touch devices. This
// object is published as a global during start up, or when such a
// device is hot plugged.  A seat typically has a pointer and
// maintains a keyboard focus and a pointer focus.
type Seat struct {
    id ObjectID
    l  SeatListener
	c  *Client
}

func (c *Client) NewSeat() *Seat {
	o := &Seat{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Seat) ID() ObjectID {
	return this.id
}

func (this *Seat) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeSeatCapabilities:
		if this.l == nil {
			fmt.Println("ignoring Capabilities event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			capabilities := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.Capabilities(capabilities)
        }
	case opCodeSeatName:
		if this.l == nil {
			fmt.Println("ignoring Name event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			len = int(hostByteOrder.Uint32(payload[0+off:4+off]))
			name := string(payload[4+off:4+len+off-1])
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}

			this.l.Name(name)
        }
	
    }
}

func (this *Seat) SetListener(listener SeatListener) {
    this.l = listener
}

// The ID provided will be initialized to the wl_pointer interface
// for this seat.
// 
// This request only takes effect if the seat has the pointer
// capability, or has had the pointer capability in the past.
// It is a protocol violation to issue this request on a seat that has
// never had the pointer capability.
func (this *Seat) GetPointer() (*Pointer, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewPointer()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSeatGetPointer)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}
// The ID provided will be initialized to the wl_keyboard interface
// for this seat.
// 
// This request only takes effect if the seat has the keyboard
// capability, or has had the keyboard capability in the past.
// It is a protocol violation to issue this request on a seat that has
// never had the keyboard capability.
func (this *Seat) GetKeyboard() (*Keyboard, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewKeyboard()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSeatGetKeyboard)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}
// The ID provided will be initialized to the wl_touch interface
// for this seat.
// 
// This request only takes effect if the seat has the touch
// capability, or has had the touch capability in the past.
// It is a protocol violation to issue this request on a seat that has
// never had the touch capability.
func (this *Seat) GetTouch() (*Touch, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewTouch()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSeatGetTouch)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}
// Using this request a client can tell the server that it is not going to
// use the seat object anymore.
func (this *Seat) Release() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSeatRelease)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    PointerErrorRole = 0 // given wl_surface has another role
)

const (
    PointerButtonStateReleased = 0 // the button is not pressed
    PointerButtonStatePressed = 1 // the button is pressed
)

const (
    PointerAxisVerticalScroll = 0 // vertical axis
    PointerAxisHorizontalScroll = 1 // horizontal axis
)

const (
    PointerAxisSourceWheel = 0 // a physical wheel rotation
    PointerAxisSourceFinger = 1 // finger on a touch surface
    PointerAxisSourceContinuous = 2 // continuous coordinate space
    PointerAxisSourceWheelTilt = 3 // a physical wheel tilt
)

const (
    opCodePointerEnter = 0
    opCodePointerLeave = 1
    opCodePointerMotion = 2
    opCodePointerButton = 3
    opCodePointerAxis = 4
    opCodePointerFrame = 5
    opCodePointerAxisSource = 6
    opCodePointerAxisStop = 7
    opCodePointerAxisDiscrete = 8
)

const (
	opCodePointerSetCursor = 0
	opCodePointerRelease = 1
)

// Pointer Events
//
// Enter
// Notification that this seat's pointer is focused on a certain
// surface.
// 
// When a seat's focus enters a surface, the pointer image
// is undefined and a client should respond to this event by setting
// an appropriate pointer image with the set_cursor request.
//
// Leave
// Notification that this seat's pointer is no longer focused on
// a certain surface.
// 
// The leave notification is sent before the enter notification
// for the new focus.
//
// Motion
// Notification of pointer location change. The arguments
// surface_x and surface_y are the location relative to the
// focused surface.
//
// Button
// Mouse button click and release notifications.
// 
// The location of the click is given by the last motion or
// enter event.
// The time argument is a timestamp with millisecond
// granularity, with an undefined base.
// 
// The button is a button code as defined in the Linux kernel's
// linux/input-event-codes.h header file, e.g. BTN_LEFT.
// 
// Any 16-bit button code value is reserved for future additions to the
// kernel's event code list. All other button codes above 0xFFFF are
// currently undefined but may be used in future versions of this
// protocol.
//
// Axis
// Scroll and other axis notifications.
// 
// For scroll events (vertical and horizontal scroll axes), the
// value parameter is the length of a vector along the specified
// axis in a coordinate space identical to those of motion events,
// representing a relative movement along the specified axis.
// 
// For devices that support movements non-parallel to axes multiple
// axis events will be emitted.
// 
// When applicable, for example for touch pads, the server can
// choose to emit scroll events where the motion vector is
// equivalent to a motion event vector.
// 
// When applicable, a client can transform its content relative to the
// scroll distance.
//
// Frame
// Indicates the end of a set of events that logically belong together.
// A client is expected to accumulate the data in all events within the
// frame before proceeding.
// 
// All wl_pointer events before a wl_pointer.frame event belong
// logically together. For example, in a diagonal scroll motion the
// compositor will send an optional wl_pointer.axis_source event, two
// wl_pointer.axis events (horizontal and vertical) and finally a
// wl_pointer.frame event. The client may use this information to
// calculate a diagonal vector for scrolling.
// 
// When multiple wl_pointer.axis events occur within the same frame,
// the motion vector is the combined motion of all events.
// When a wl_pointer.axis and a wl_pointer.axis_stop event occur within
// the same frame, this indicates that axis movement in one axis has
// stopped but continues in the other axis.
// When multiple wl_pointer.axis_stop events occur within the same
// frame, this indicates that these axes stopped in the same instance.
// 
// A wl_pointer.frame event is sent for every logical event group,
// even if the group only contains a single wl_pointer event.
// Specifically, a client may get a sequence: motion, frame, button,
// frame, axis, frame, axis_stop, frame.
// 
// The wl_pointer.enter and wl_pointer.leave events are logical events
// generated by the compositor and not the hardware. These events are
// also grouped by a wl_pointer.frame. When a pointer moves from one
// surface to another, a compositor should group the
// wl_pointer.leave event within the same wl_pointer.frame.
// However, a client must not rely on wl_pointer.leave and
// wl_pointer.enter being in the same wl_pointer.frame.
// Compositor-specific policies may require the wl_pointer.leave and
// wl_pointer.enter event being split across multiple wl_pointer.frame
// groups.
//
// AxisSource
// Source information for scroll and other axes.
// 
// This event does not occur on its own. It is sent before a
// wl_pointer.frame event and carries the source information for
// all events within that frame.
// 
// The source specifies how this event was generated. If the source is
// wl_pointer.axis_source.finger, a wl_pointer.axis_stop event will be
// sent when the user lifts the finger off the device.
// 
// If the source is wl_pointer.axis_source.wheel,
// wl_pointer.axis_source.wheel_tilt or
// wl_pointer.axis_source.continuous, a wl_pointer.axis_stop event may
// or may not be sent. Whether a compositor sends an axis_stop event
// for these sources is hardware-specific and implementation-dependent;
// clients must not rely on receiving an axis_stop event for these
// scroll sources and should treat scroll sequences from these scroll
// sources as unterminated by default.
// 
// This event is optional. If the source is unknown for a particular
// axis event sequence, no event is sent.
// Only one wl_pointer.axis_source event is permitted per frame.
// 
// The order of wl_pointer.axis_discrete and wl_pointer.axis_source is
// not guaranteed.
//
// AxisStop
// Stop notification for scroll and other axes.
// 
// For some wl_pointer.axis_source types, a wl_pointer.axis_stop event
// is sent to notify a client that the axis sequence has terminated.
// This enables the client to implement kinetic scrolling.
// See the wl_pointer.axis_source documentation for information on when
// this event may be generated.
// 
// Any wl_pointer.axis events with the same axis_source after this
// event should be considered as the start of a new axis motion.
// 
// The timestamp is to be interpreted identical to the timestamp in the
// wl_pointer.axis event. The timestamp value may be the same as a
// preceding wl_pointer.axis event.
//
// AxisDiscrete
// Discrete step information for scroll and other axes.
// 
// This event carries the axis value of the wl_pointer.axis event in
// discrete steps (e.g. mouse wheel clicks).
// 
// This event does not occur on its own, it is coupled with a
// wl_pointer.axis event that represents this axis value on a
// continuous scale. The protocol guarantees that each axis_discrete
// event is always followed by exactly one axis event with the same
// axis number within the same wl_pointer.frame. Note that the protocol
// allows for other events to occur between the axis_discrete and
// its coupled axis event, including other axis_discrete or axis
// events.
// 
// This event is optional; continuous scrolling devices
// like two-finger scrolling on touchpads do not have discrete
// steps and do not generate this event.
// 
// The discrete value carries the directional information. e.g. a value
// of -2 is two steps towards the negative direction of this axis.
// 
// The axis number is identical to the axis number in the associated
// axis event.
// 
// The order of wl_pointer.axis_discrete and wl_pointer.axis_source is
// not guaranteed.
type PointerListener interface {
    Enter(serial uint32, surface uint32, surfaceX float64, surfaceY float64)
    Leave(serial uint32, surface uint32)
    Motion(time uint32, surfaceX float64, surfaceY float64)
    Button(serial uint32, time uint32, button uint32, state uint32)
    Axis(time uint32, axis uint32, value float64)
    Frame()
    AxisSource(axisSource uint32)
    AxisStop(time uint32, axis uint32)
    AxisDiscrete(axis uint32, discrete int32)
}

// The wl_pointer interface represents one or more input devices,
// such as mice, which control the pointer location and pointer_focus
// of a seat.
// 
// The wl_pointer interface generates motion, enter and leave
// events for the surfaces that the pointer is located over,
// and button and axis events for button presses, button releases
// and scrolling.
type Pointer struct {
    id ObjectID
    l  PointerListener
	c  *Client
}

func (c *Client) NewPointer() *Pointer {
	o := &Pointer{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Pointer) ID() ObjectID {
	return this.id
}

func (this *Pointer) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodePointerEnter:
		if this.l == nil {
			fmt.Println("ignoring Enter event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			surface := hostByteOrder.Uint32(payload[4+off:8+off])
			surfaceX := fixedToFloat64(int32(hostByteOrder.Uint32(payload[8+off:12+off])))
			surfaceY := fixedToFloat64(int32(hostByteOrder.Uint32(payload[12+off:16+off])))

			this.l.Enter(serial, surface, surfaceX, surfaceY)
        }
	case opCodePointerLeave:
		if this.l == nil {
			fmt.Println("ignoring Leave event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			surface := hostByteOrder.Uint32(payload[4+off:8+off])

			this.l.Leave(serial, surface)
        }
	case opCodePointerMotion:
		if this.l == nil {
			fmt.Println("ignoring Motion event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			time := hostByteOrder.Uint32(payload[0+off:4+off])
			surfaceX := fixedToFloat64(int32(hostByteOrder.Uint32(payload[4+off:8+off])))
			surfaceY := fixedToFloat64(int32(hostByteOrder.Uint32(payload[8+off:12+off])))

			this.l.Motion(time, surfaceX, surfaceY)
        }
	case opCodePointerButton:
		if this.l == nil {
			fmt.Println("ignoring Button event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			time := hostByteOrder.Uint32(payload[4+off:8+off])
			button := hostByteOrder.Uint32(payload[8+off:12+off])
			state := hostByteOrder.Uint32(payload[12+off:16+off])

			this.l.Button(serial, time, button, state)
        }
	case opCodePointerAxis:
		if this.l == nil {
			fmt.Println("ignoring Axis event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			time := hostByteOrder.Uint32(payload[0+off:4+off])
			axis := hostByteOrder.Uint32(payload[4+off:8+off])
			value := fixedToFloat64(int32(hostByteOrder.Uint32(payload[8+off:12+off])))

			this.l.Axis(time, axis, value)
        }
	case opCodePointerFrame:
		if this.l == nil {
			fmt.Println("ignoring Frame event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.Frame()
        }
	case opCodePointerAxisSource:
		if this.l == nil {
			fmt.Println("ignoring AxisSource event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			axisSource := hostByteOrder.Uint32(payload[0+off:4+off])

			this.l.AxisSource(axisSource)
        }
	case opCodePointerAxisStop:
		if this.l == nil {
			fmt.Println("ignoring AxisStop event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			time := hostByteOrder.Uint32(payload[0+off:4+off])
			axis := hostByteOrder.Uint32(payload[4+off:8+off])

			this.l.AxisStop(time, axis)
        }
	case opCodePointerAxisDiscrete:
		if this.l == nil {
			fmt.Println("ignoring AxisDiscrete event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			axis := hostByteOrder.Uint32(payload[0+off:4+off])
			discrete := int32(hostByteOrder.Uint32(payload[4+off:8+off]))

			this.l.AxisDiscrete(axis, discrete)
        }
	
    }
}

func (this *Pointer) SetListener(listener PointerListener) {
    this.l = listener
}

// Set the pointer surface, i.e., the surface that contains the
// pointer image (cursor). This request gives the surface the role
// of a cursor. If the surface already has another role, it raises
// a protocol error.
// 
// The cursor actually changes only if the pointer
// focus for this device is one of the requesting client's surfaces
// or the surface parameter is the current pointer surface. If
// there was a previous surface set with this request it is
// replaced. If surface is NULL, the pointer image is hidden.
// 
// The parameters hotspot_x and hotspot_y define the position of
// the pointer surface relative to the pointer location. Its
// top-left corner is always at (x, y) - (hotspot_x, hotspot_y),
// where (x, y) are the coordinates of the pointer location, in
// surface-local coordinates.
// 
// On surface.attach requests to the pointer surface, hotspot_x
// and hotspot_y are decremented by the x and y parameters
// passed to the request. Attach must be confirmed by
// wl_surface.commit as usual.
// 
// The hotspot can also be updated by passing the currently set
// pointer surface to this request with new values for hotspot_x
// and hotspot_y.
// 
// The current and pending input regions of the wl_surface are
// cleared, and wl_surface.set_input_region is ignored until the
// wl_surface is no longer used as the cursor. When the use as a
// cursor ends, the current and pending input regions become
// undefined, and the wl_surface is unmapped.
func (this *Pointer) SetCursor(serial uint32, surface uint32, hotspotX int32, hotspotY int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, serial)
	binary.Write(this.c.buf, hostByteOrder, surface)
	binary.Write(this.c.buf, hostByteOrder, hotspotX)
	binary.Write(this.c.buf, hostByteOrder, hotspotY)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodePointerSetCursor)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Using this request a client can tell the server that it is not going to
// use the pointer object anymore.
// 
// This request destroys the pointer proxy object, so clients must not call
// wl_pointer_destroy() after using this request.
func (this *Pointer) Release() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodePointerRelease)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    KeyboardKeymapFormatNoKeymap = 0 // no keymap; client must understand how to interpret the raw keycode
    KeyboardKeymapFormatXkbV1 = 1 // libxkbcommon compatible; to determine the xkb keycode, clients must add 8 to the key event keycode
)

const (
    KeyboardKeyStateReleased = 0 // key is not pressed
    KeyboardKeyStatePressed = 1 // key is pressed
)

const (
    opCodeKeyboardKeymap = 0
    opCodeKeyboardEnter = 1
    opCodeKeyboardLeave = 2
    opCodeKeyboardKey = 3
    opCodeKeyboardModifiers = 4
    opCodeKeyboardRepeatInfo = 5
)

const (
	opCodeKeyboardRelease = 0
)

// Keyboard Events
//
// Keymap
// This event provides a file descriptor to the client which can be
// memory-mapped to provide a keyboard mapping description.
//
// Enter
// Notification that this seat's keyboard focus is on a certain
// surface.
//
// Leave
// Notification that this seat's keyboard focus is no longer on
// a certain surface.
// 
// The leave notification is sent before the enter notification
// for the new focus.
//
// Key
// A key was pressed or released.
// The time argument is a timestamp with millisecond
// granularity, with an undefined base.
//
// Modifiers
// Notifies clients that the modifier and/or group state has
// changed, and it should update its local state.
//
// RepeatInfo
// Informs the client about the keyboard's repeat rate and delay.
// 
// This event is sent as soon as the wl_keyboard object has been created,
// and is guaranteed to be received by the client before any key press
// event.
// 
// Negative values for either rate or delay are illegal. A rate of zero
// will disable any repeating (regardless of the value of delay).
// 
// This event can be sent later on as well with a new value if necessary,
// so clients should continue listening for the event past the creation
// of wl_keyboard.
type KeyboardListener interface {
    Keymap(format uint32, fd *os.File, size uint32)
    Enter(serial uint32, surface uint32, keys []byte)
    Leave(serial uint32, surface uint32)
    Key(serial uint32, time uint32, key uint32, state uint32)
    Modifiers(serial uint32, modsDepressed uint32, modsLatched uint32, modsLocked uint32, group uint32)
    RepeatInfo(rate int32, delay int32)
}

// The wl_keyboard interface represents one or more keyboards
// associated with a seat.
type Keyboard struct {
    id ObjectID
    l  KeyboardListener
	c  *Client
}

func (c *Client) NewKeyboard() *Keyboard {
	o := &Keyboard{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Keyboard) ID() ObjectID {
	return this.id
}

func (this *Keyboard) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeKeyboardKeymap:
		if this.l == nil {
			fmt.Println("ignoring Keymap event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			format := hostByteOrder.Uint32(payload[0+off:4+off])
			fd := file
			size := hostByteOrder.Uint32(payload[4+off:8+off])

			this.l.Keymap(format, fd, size)
        }
	case opCodeKeyboardEnter:
		if this.l == nil {
			fmt.Println("ignoring Enter event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			surface := hostByteOrder.Uint32(payload[4+off:8+off])
			len = int(hostByteOrder.Uint32(payload[8+off:12+off]))
			keys := payload[12+off:12+len+off]
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}

			this.l.Enter(serial, surface, keys)
        }
	case opCodeKeyboardLeave:
		if this.l == nil {
			fmt.Println("ignoring Leave event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			surface := hostByteOrder.Uint32(payload[4+off:8+off])

			this.l.Leave(serial, surface)
        }
	case opCodeKeyboardKey:
		if this.l == nil {
			fmt.Println("ignoring Key event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			time := hostByteOrder.Uint32(payload[4+off:8+off])
			key := hostByteOrder.Uint32(payload[8+off:12+off])
			state := hostByteOrder.Uint32(payload[12+off:16+off])

			this.l.Key(serial, time, key, state)
        }
	case opCodeKeyboardModifiers:
		if this.l == nil {
			fmt.Println("ignoring Modifiers event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			modsDepressed := hostByteOrder.Uint32(payload[4+off:8+off])
			modsLatched := hostByteOrder.Uint32(payload[8+off:12+off])
			modsLocked := hostByteOrder.Uint32(payload[12+off:16+off])
			group := hostByteOrder.Uint32(payload[16+off:20+off])

			this.l.Modifiers(serial, modsDepressed, modsLatched, modsLocked, group)
        }
	case opCodeKeyboardRepeatInfo:
		if this.l == nil {
			fmt.Println("ignoring RepeatInfo event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			rate := int32(hostByteOrder.Uint32(payload[0+off:4+off]))
			delay := int32(hostByteOrder.Uint32(payload[4+off:8+off]))

			this.l.RepeatInfo(rate, delay)
        }
	
    }
}

func (this *Keyboard) SetListener(listener KeyboardListener) {
    this.l = listener
}

func (this *Keyboard) Release() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeKeyboardRelease)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    opCodeTouchDown = 0
    opCodeTouchUp = 1
    opCodeTouchMotion = 2
    opCodeTouchFrame = 3
    opCodeTouchCancel = 4
    opCodeTouchShape = 5
    opCodeTouchOrientation = 6
)

const (
	opCodeTouchRelease = 0
)

// Touch Events
//
// Down
// A new touch point has appeared on the surface. This touch point is
// assigned a unique ID. Future events from this touch point reference
// this ID. The ID ceases to be valid after a touch up event and may be
// reused in the future.
//
// Up
// The touch point has disappeared. No further events will be sent for
// this touch point and the touch point's ID is released and may be
// reused in a future touch down event.
//
// Motion
// A touch point has changed coordinates.
//
// Frame
// Indicates the end of a set of events that logically belong together.
// A client is expected to accumulate the data in all events within the
// frame before proceeding.
// 
// A wl_touch.frame terminates at least one event but otherwise no
// guarantee is provided about the set of events within a frame. A client
// must assume that any state not updated in a frame is unchanged from the
// previously known state.
//
// Cancel
// Sent if the compositor decides the touch stream is a global
// gesture. No further events are sent to the clients from that
// particular gesture. Touch cancellation applies to all touch points
// currently active on this client's surface. The client is
// responsible for finalizing the touch points, future touch points on
// this surface may reuse the touch point ID.
//
// Shape
// Sent when a touchpoint has changed its shape.
// 
// This event does not occur on its own. It is sent before a
// wl_touch.frame event and carries the new shape information for
// any previously reported, or new touch points of that frame.
// 
// Other events describing the touch point such as wl_touch.down,
// wl_touch.motion or wl_touch.orientation may be sent within the
// same wl_touch.frame. A client should treat these events as a single
// logical touch point update. The order of wl_touch.shape,
// wl_touch.orientation and wl_touch.motion is not guaranteed.
// A wl_touch.down event is guaranteed to occur before the first
// wl_touch.shape event for this touch ID but both events may occur within
// the same wl_touch.frame.
// 
// A touchpoint shape is approximated by an ellipse through the major and
// minor axis length. The major axis length describes the longer diameter
// of the ellipse, while the minor axis length describes the shorter
// diameter. Major and minor are orthogonal and both are specified in
// surface-local coordinates. The center of the ellipse is always at the
// touchpoint location as reported by wl_touch.down or wl_touch.move.
// 
// This event is only sent by the compositor if the touch device supports
// shape reports. The client has to make reasonable assumptions about the
// shape if it did not receive this event.
//
// Orientation
// Sent when a touchpoint has changed its orientation.
// 
// This event does not occur on its own. It is sent before a
// wl_touch.frame event and carries the new shape information for
// any previously reported, or new touch points of that frame.
// 
// Other events describing the touch point such as wl_touch.down,
// wl_touch.motion or wl_touch.shape may be sent within the
// same wl_touch.frame. A client should treat these events as a single
// logical touch point update. The order of wl_touch.shape,
// wl_touch.orientation and wl_touch.motion is not guaranteed.
// A wl_touch.down event is guaranteed to occur before the first
// wl_touch.orientation event for this touch ID but both events may occur
// within the same wl_touch.frame.
// 
// The orientation describes the clockwise angle of a touchpoint's major
// axis to the positive surface y-axis and is normalized to the -180 to
// +180 degree range. The granularity of orientation depends on the touch
// device, some devices only support binary rotation values between 0 and
// 90 degrees.
// 
// This event is only sent by the compositor if the touch device supports
// orientation reports.
type TouchListener interface {
    Down(serial uint32, time uint32, surface uint32, id int32, x float64, y float64)
    Up(serial uint32, time uint32, id int32)
    Motion(time uint32, id int32, x float64, y float64)
    Frame()
    Cancel()
    Shape(id int32, major float64, minor float64)
    Orientation(id int32, orientation float64)
}

// The wl_touch interface represents a touchscreen
// associated with a seat.
// 
// Touch interactions can consist of one or more contacts.
// For each contact, a series of events is generated, starting
// with a down event, followed by zero or more motion events,
// and ending with an up event. Events relating to the same
// contact point can be identified by the ID of the sequence.
type Touch struct {
    id ObjectID
    l  TouchListener
	c  *Client
}

func (c *Client) NewTouch() *Touch {
	o := &Touch{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Touch) ID() ObjectID {
	return this.id
}

func (this *Touch) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeTouchDown:
		if this.l == nil {
			fmt.Println("ignoring Down event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			time := hostByteOrder.Uint32(payload[4+off:8+off])
			surface := hostByteOrder.Uint32(payload[8+off:12+off])
			id := int32(hostByteOrder.Uint32(payload[12+off:16+off]))
			x := fixedToFloat64(int32(hostByteOrder.Uint32(payload[16+off:20+off])))
			y := fixedToFloat64(int32(hostByteOrder.Uint32(payload[20+off:24+off])))

			this.l.Down(serial, time, surface, id, x, y)
        }
	case opCodeTouchUp:
		if this.l == nil {
			fmt.Println("ignoring Up event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			serial := hostByteOrder.Uint32(payload[0+off:4+off])
			time := hostByteOrder.Uint32(payload[4+off:8+off])
			id := int32(hostByteOrder.Uint32(payload[8+off:12+off]))

			this.l.Up(serial, time, id)
        }
	case opCodeTouchMotion:
		if this.l == nil {
			fmt.Println("ignoring Motion event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			time := hostByteOrder.Uint32(payload[0+off:4+off])
			id := int32(hostByteOrder.Uint32(payload[4+off:8+off]))
			x := fixedToFloat64(int32(hostByteOrder.Uint32(payload[8+off:12+off])))
			y := fixedToFloat64(int32(hostByteOrder.Uint32(payload[12+off:16+off])))

			this.l.Motion(time, id, x, y)
        }
	case opCodeTouchFrame:
		if this.l == nil {
			fmt.Println("ignoring Frame event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.Frame()
        }
	case opCodeTouchCancel:
		if this.l == nil {
			fmt.Println("ignoring Cancel event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.Cancel()
        }
	case opCodeTouchShape:
		if this.l == nil {
			fmt.Println("ignoring Shape event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			id := int32(hostByteOrder.Uint32(payload[0+off:4+off]))
			major := fixedToFloat64(int32(hostByteOrder.Uint32(payload[4+off:8+off])))
			minor := fixedToFloat64(int32(hostByteOrder.Uint32(payload[8+off:12+off])))

			this.l.Shape(id, major, minor)
        }
	case opCodeTouchOrientation:
		if this.l == nil {
			fmt.Println("ignoring Orientation event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			id := int32(hostByteOrder.Uint32(payload[0+off:4+off]))
			orientation := fixedToFloat64(int32(hostByteOrder.Uint32(payload[4+off:8+off])))

			this.l.Orientation(id, orientation)
        }
	
    }
}

func (this *Touch) SetListener(listener TouchListener) {
    this.l = listener
}

func (this *Touch) Release() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeTouchRelease)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    OutputSubpixelUnknown = 0 // unknown geometry
    OutputSubpixelNone = 1 // no geometry
    OutputSubpixelHorizontalRgb = 2 // horizontal RGB
    OutputSubpixelHorizontalBgr = 3 // horizontal BGR
    OutputSubpixelVerticalRgb = 4 // vertical RGB
    OutputSubpixelVerticalBgr = 5 // vertical BGR
)

const (
    OutputTransformNormal = 0 // no transform
    OutputTransform90 = 1 // 90 degrees counter-clockwise
    OutputTransform180 = 2 // 180 degrees counter-clockwise
    OutputTransform270 = 3 // 270 degrees counter-clockwise
    OutputTransformFlipped = 4 // 180 degree flip around a vertical axis
    OutputTransformFlipped90 = 5 // flip and rotate 90 degrees counter-clockwise
    OutputTransformFlipped180 = 6 // flip and rotate 180 degrees counter-clockwise
    OutputTransformFlipped270 = 7 // flip and rotate 270 degrees counter-clockwise
)

const (
    OutputModeCurrent = 0x1 // indicates this is the current mode
    OutputModePreferred = 0x2 // indicates this is the preferred mode
)

const (
    opCodeOutputGeometry = 0
    opCodeOutputMode = 1
    opCodeOutputDone = 2
    opCodeOutputScale = 3
)

const (
	opCodeOutputRelease = 0
)

// Output Events
//
// Geometry
// The geometry event describes geometric properties of the output.
// The event is sent when binding to the output object and whenever
// any of the properties change.
//
// Mode
// The mode event describes an available mode for the output.
// 
// The event is sent when binding to the output object and there
// will always be one mode, the current mode.  The event is sent
// again if an output changes mode, for the mode that is now
// current.  In other words, the current mode is always the last
// mode that was received with the current flag set.
// 
// The size of a mode is given in physical hardware units of
// the output device. This is not necessarily the same as
// the output size in the global compositor space. For instance,
// the output may be scaled, as described in wl_output.scale,
// or transformed, as described in wl_output.transform.
//
// Done
// This event is sent after all other properties have been
// sent after binding to the output object and after any
// other property changes done after that. This allows
// changes to the output properties to be seen as
// atomic, even if they happen via multiple events.
//
// Scale
// This event contains scaling geometry information
// that is not in the geometry event. It may be sent after
// binding the output object or if the output scale changes
// later. If it is not sent, the client should assume a
// scale of 1.
// 
// A scale larger than 1 means that the compositor will
// automatically scale surface buffers by this amount
// when rendering. This is used for very high resolution
// displays where applications rendering at the native
// resolution would be too small to be legible.
// 
// It is intended that scaling aware clients track the
// current output of a surface, and if it is on a scaled
// output it should use wl_surface.set_buffer_scale with
// the scale of the output. That way the compositor can
// avoid scaling the surface, and the client can supply
// a higher detail image.
type OutputListener interface {
    Geometry(x int32, y int32, physicalWidth int32, physicalHeight int32, subpixel int32, make string, model string, transform int32)
    Mode(flags uint32, width int32, height int32, refresh int32)
    Done()
    Scale(factor int32)
}

// An output describes part of the compositor geometry.  The
// compositor works in the 'compositor coordinate system' and an
// output corresponds to a rectangular area in that space that is
// actually visible.  This typically corresponds to a monitor that
// displays part of the compositor space.  This object is published
// as global during start up, or when a monitor is hotplugged.
type Output struct {
    id ObjectID
    l  OutputListener
	c  *Client
}

func (c *Client) NewOutput() *Output {
	o := &Output{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Output) ID() ObjectID {
	return this.id
}

func (this *Output) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	case opCodeOutputGeometry:
		if this.l == nil {
			fmt.Println("ignoring Geometry event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			x := int32(hostByteOrder.Uint32(payload[0+off:4+off]))
			y := int32(hostByteOrder.Uint32(payload[4+off:8+off]))
			physicalWidth := int32(hostByteOrder.Uint32(payload[8+off:12+off]))
			physicalHeight := int32(hostByteOrder.Uint32(payload[12+off:16+off]))
			subpixel := int32(hostByteOrder.Uint32(payload[16+off:20+off]))
			len = int(hostByteOrder.Uint32(payload[20+off:24+off]))
			make := string(payload[24+off:24+len+off-1])
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}
			len = int(hostByteOrder.Uint32(payload[24+off:28+off]))
			model := string(payload[28+off:28+len+off-1])
			off += len
			if off % 4 != 0 {
				off += 4 - off % 4
			}
			transform := int32(hostByteOrder.Uint32(payload[28+off:32+off]))

			this.l.Geometry(x, y, physicalWidth, physicalHeight, subpixel, make, model, transform)
        }
	case opCodeOutputMode:
		if this.l == nil {
			fmt.Println("ignoring Mode event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			flags := hostByteOrder.Uint32(payload[0+off:4+off])
			width := int32(hostByteOrder.Uint32(payload[4+off:8+off]))
			height := int32(hostByteOrder.Uint32(payload[8+off:12+off]))
			refresh := int32(hostByteOrder.Uint32(payload[12+off:16+off]))

			this.l.Mode(flags, width, height, refresh)
        }
	case opCodeOutputDone:
		if this.l == nil {
			fmt.Println("ignoring Done event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len

			this.l.Done()
        }
	case opCodeOutputScale:
		if this.l == nil {
			fmt.Println("ignoring Scale event: no listener")
		} else {
			off, len := 0, 0
			_, _ = off, len
			factor := int32(hostByteOrder.Uint32(payload[0+off:4+off]))

			this.l.Scale(factor)
        }
	
    }
}

func (this *Output) SetListener(listener OutputListener) {
    this.l = listener
}

// Using this request a client can tell the server that it is not going to
// use the output object anymore.
func (this *Output) Release() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeOutputRelease)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
)

const (
	opCodeRegionDestroy = 0
	opCodeRegionAdd = 1
	opCodeRegionSubtract = 2
)

// Region Events
type RegionListener interface {
}

// A region object describes an area.
// 
// Region objects are used to describe the opaque and input
// regions of a surface.
type Region struct {
    id ObjectID
    l  RegionListener
	c  *Client
}

func (c *Client) NewRegion() *Region {
	o := &Region{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Region) ID() ObjectID {
	return this.id
}

func (this *Region) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	
    }
}

func (this *Region) SetListener(listener RegionListener) {
    this.l = listener
}

// Destroy the region.  This will invalidate the object ID.
func (this *Region) Destroy() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeRegionDestroy)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Add the specified rectangle to the region.
func (this *Region) Add(x int32, y int32, width int32, height int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, x)
	binary.Write(this.c.buf, hostByteOrder, y)
	binary.Write(this.c.buf, hostByteOrder, width)
	binary.Write(this.c.buf, hostByteOrder, height)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeRegionAdd)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Subtract the specified rectangle from the region.
func (this *Region) Subtract(x int32, y int32, width int32, height int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, x)
	binary.Write(this.c.buf, hostByteOrder, y)
	binary.Write(this.c.buf, hostByteOrder, width)
	binary.Write(this.c.buf, hostByteOrder, height)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeRegionSubtract)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}

const (
    SubcompositorErrorBadSurface = 0 // the to-be sub-surface is invalid
)

const (
)

const (
	opCodeSubcompositorDestroy = 0
	opCodeSubcompositorGetSubsurface = 1
)

// Subcompositor Events
type SubcompositorListener interface {
}

// The global interface exposing sub-surface compositing capabilities.
// A wl_surface, that has sub-surfaces associated, is called the
// parent surface. Sub-surfaces can be arbitrarily nested and create
// a tree of sub-surfaces.
// 
// The root surface in a tree of sub-surfaces is the main
// surface. The main surface cannot be a sub-surface, because
// sub-surfaces must always have a parent.
// 
// A main surface with its sub-surfaces forms a (compound) window.
// For window management purposes, this set of wl_surface objects is
// to be considered as a single window, and it should also behave as
// such.
// 
// The aim of sub-surfaces is to offload some of the compositing work
// within a window from clients to the compositor. A prime example is
// a video player with decorations and video in separate wl_surface
// objects. This should allow the compositor to pass YUV video buffer
// processing to dedicated overlay hardware when possible.
type Subcompositor struct {
    id ObjectID
    l  SubcompositorListener
	c  *Client
}

func (c *Client) NewSubcompositor() *Subcompositor {
	o := &Subcompositor{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Subcompositor) ID() ObjectID {
	return this.id
}

func (this *Subcompositor) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	
    }
}

func (this *Subcompositor) SetListener(listener SubcompositorListener) {
    this.l = listener
}

// Informs the server that the client will not be using this
// protocol object anymore. This does not affect any other
// objects, wl_subsurface objects included.
func (this *Subcompositor) Destroy() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSubcompositorDestroy)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Create a sub-surface interface for the given surface, and
// associate it with the given parent surface. This turns a
// plain wl_surface into a sub-surface.
// 
// The to-be sub-surface must not already have another role, and it
// must not have an existing wl_subsurface object. Otherwise a protocol
// error is raised.
func (this *Subcompositor) GetSubsurface(surface uint32, parent uint32) (*Subsurface, error) {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := this.c.NewSubsurface()
	binary.Write(this.c.buf, hostByteOrder, ret.ID())
	binary.Write(this.c.buf, hostByteOrder, surface)
	binary.Write(this.c.buf, hostByteOrder, parent)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSubcompositorGetSubsurface)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return ret, nil
}

const (
    SubsurfaceErrorBadSurface = 0 // wl_surface is not a sibling or the parent
)

const (
)

const (
	opCodeSubsurfaceDestroy = 0
	opCodeSubsurfaceSetPosition = 1
	opCodeSubsurfacePlaceAbove = 2
	opCodeSubsurfacePlaceBelow = 3
	opCodeSubsurfaceSetSync = 4
	opCodeSubsurfaceSetDesync = 5
)

// Subsurface Events
type SubsurfaceListener interface {
}

// An additional interface to a wl_surface object, which has been
// made a sub-surface. A sub-surface has one parent surface. A
// sub-surface's size and position are not limited to that of the parent.
// Particularly, a sub-surface is not automatically clipped to its
// parent's area.
// 
// A sub-surface becomes mapped, when a non-NULL wl_buffer is applied
// and the parent surface is mapped. The order of which one happens
// first is irrelevant. A sub-surface is hidden if the parent becomes
// hidden, or if a NULL wl_buffer is applied. These rules apply
// recursively through the tree of surfaces.
// 
// The behaviour of a wl_surface.commit request on a sub-surface
// depends on the sub-surface's mode. The possible modes are
// synchronized and desynchronized, see methods
// wl_subsurface.set_sync and wl_subsurface.set_desync. Synchronized
// mode caches the wl_surface state to be applied when the parent's
// state gets applied, and desynchronized mode applies the pending
// wl_surface state directly. A sub-surface is initially in the
// synchronized mode.
// 
// Sub-surfaces have also other kind of state, which is managed by
// wl_subsurface requests, as opposed to wl_surface requests. This
// state includes the sub-surface position relative to the parent
// surface (wl_subsurface.set_position), and the stacking order of
// the parent and its sub-surfaces (wl_subsurface.place_above and
// .place_below). This state is applied when the parent surface's
// wl_surface state is applied, regardless of the sub-surface's mode.
// As the exception, set_sync and set_desync are effective immediately.
// 
// The main surface can be thought to be always in desynchronized mode,
// since it does not have a parent in the sub-surfaces sense.
// 
// Even if a sub-surface is in desynchronized mode, it will behave as
// in synchronized mode, if its parent surface behaves as in
// synchronized mode. This rule is applied recursively throughout the
// tree of surfaces. This means, that one can set a sub-surface into
// synchronized mode, and then assume that all its child and grand-child
// sub-surfaces are synchronized, too, without explicitly setting them.
// 
// If the wl_surface associated with the wl_subsurface is destroyed, the
// wl_subsurface object becomes inert. Note, that destroying either object
// takes effect immediately. If you need to synchronize the removal
// of a sub-surface to the parent surface update, unmap the sub-surface
// first by attaching a NULL wl_buffer, update parent, and then destroy
// the sub-surface.
// 
// If the parent wl_surface object is destroyed, the sub-surface is
// unmapped.
type Subsurface struct {
    id ObjectID
    l  SubsurfaceListener
	c  *Client
}

func (c *Client) NewSubsurface() *Subsurface {
	o := &Subsurface{
		id: GetNewID(),
		c: c,
	}
	c.objects[o.ID()] = o
	return o
}

func (this *Subsurface) ID() ObjectID {
	return this.id
}

func (this *Subsurface) Dispatch(opCode uint16, payload []byte, file *os.File) {
	switch opCode {
	
    }
}

func (this *Subsurface) SetListener(listener SubsurfaceListener) {
    this.l = listener
}

// The sub-surface interface is removed from the wl_surface object
// that was turned into a sub-surface with a
// wl_subcompositor.get_subsurface request. The wl_surface's association
// to the parent is deleted, and the wl_surface loses its role as
// a sub-surface. The wl_surface is unmapped.
func (this *Subsurface) Destroy() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSubsurfaceDestroy)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This schedules a sub-surface position change.
// The sub-surface will be moved so that its origin (top left
// corner pixel) will be at the location x, y of the parent surface
// coordinate system. The coordinates are not restricted to the parent
// surface area. Negative values are allowed.
// 
// The scheduled coordinates will take effect whenever the state of the
// parent surface is applied. When this happens depends on whether the
// parent surface is in synchronized mode or not. See
// wl_subsurface.set_sync and wl_subsurface.set_desync for details.
// 
// If more than one set_position request is invoked by the client before
// the commit of the parent surface, the position of a new request always
// replaces the scheduled position from any previous request.
// 
// The initial position is 0, 0.
func (this *Subsurface) SetPosition(x int32, y int32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, x)
	binary.Write(this.c.buf, hostByteOrder, y)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSubsurfaceSetPosition)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// This sub-surface is taken from the stack, and put back just
// above the reference surface, changing the z-order of the sub-surfaces.
// The reference surface must be one of the sibling surfaces, or the
// parent surface. Using any other surface, including this sub-surface,
// will cause a protocol error.
// 
// The z-order is double-buffered. Requests are handled in order and
// applied immediately to a pending state. The final pending state is
// copied to the active state the next time the state of the parent
// surface is applied. When this happens depends on whether the parent
// surface is in synchronized mode or not. See wl_subsurface.set_sync and
// wl_subsurface.set_desync for details.
// 
// A new sub-surface is initially added as the top-most in the stack
// of its siblings and parent.
func (this *Subsurface) PlaceAbove(sibling uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, sibling)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSubsurfacePlaceAbove)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// The sub-surface is placed just below the reference surface.
// See wl_subsurface.place_above.
func (this *Subsurface) PlaceBelow(sibling uint32) error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, sibling)

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSubsurfacePlaceBelow)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Change the commit behaviour of the sub-surface to synchronized
// mode, also described as the parent dependent mode.
// 
// In synchronized mode, wl_surface.commit on a sub-surface will
// accumulate the committed state in a cache, but the state will
// not be applied and hence will not change the compositor output.
// The cached state is applied to the sub-surface immediately after
// the parent surface's state is applied. This ensures atomic
// updates of the parent and all its synchronized sub-surfaces.
// Applying the cached state will invalidate the cache, so further
// parent surface commits do not (re-)apply old state.
// 
// See wl_subsurface for the recursive effect of this mode.
func (this *Subsurface) SetSync() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSubsurfaceSetSync)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
// Change the commit behaviour of the sub-surface to desynchronized
// mode, also described as independent or freely running mode.
// 
// In desynchronized mode, wl_surface.commit on a sub-surface will
// apply the pending state directly, without caching, as happens
// normally with a wl_surface. Calling wl_surface.commit on the
// parent surface has no effect on the sub-surface's wl_surface
// state. This mode allows a sub-surface to be updated on its own.
// 
// If cached state exists when wl_surface.commit is called in
// desynchronized mode, the pending state is added to the cached
// state, and applied as a whole. This invalidates the cache.
// 
// Note: even if a sub-surface is set to desynchronized, a parent
// sub-surface may override it to behave as synchronized. For details,
// see wl_subsurface.
// 
// If a surface's parent surface behaves as desynchronized, then
// the cached state is applied on set_desync.
func (this *Subsurface) SetDesync() error {
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.ID())
	binary.Write(this.c.buf, hostByteOrder, uint32(0))

	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len()) << 16 | opCodeSubsurfaceSetDesync)
	fmt.Println("Sending: ")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.conn.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
    return nil
}
