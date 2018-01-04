package wlp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

const (
	ZxdgShellV6ErrorRole                = 0 // given wl_surface has another role
	ZxdgShellV6ErrorDefunctSurfaces     = 1 // xdg_shell was destroyed before children
	ZxdgShellV6ErrorNotTheTopmostPopup  = 2 // the client tried to map or destroy a non-topmost popup
	ZxdgShellV6ErrorInvalidPopupParent  = 3 // the client specified an invalid popup parent surface
	ZxdgShellV6ErrorInvalidSurfaceState = 4 // the client provided an invalid surface state
	ZxdgShellV6ErrorInvalidPositioner   = 5 // the client provided an invalid positioner
)

const (
	opCodeZxdgShellV6Ping = 0
)

const (
	opCodeZxdgShellV6Destroy          = 0
	opCodeZxdgShellV6CreatePositioner = 1
	opCodeZxdgShellV6GetXdgSurface    = 2
	opCodeZxdgShellV6Pong             = 3
)

// ZxdgShellV6 Events
//
// Ping
// The ping event asks the client if it's still alive. Pass the
// serial specified in the event back to the compositor by sending
// a "pong" request back with the specified serial. See xdg_shell.ping.
//
// Compositors can use this to determine if the client is still
// alive. It's unspecified what will happen if the client doesn't
// respond to the ping request, or in what timeframe. Clients should
// try to respond in a reasonable amount of time.
//
// A compositor is free to ping in any way it wants, but a client must
// always respond to any xdg_shell object it created.
type ZxdgShellV6Listener interface {
	Ping(serial uint32)
}

// xdg_shell allows clients to turn a wl_surface into a "real window"
// which can be dragged, resized, stacked, and moved around by the
// user. Everything about this interface is suited towards traditional
// desktop environments.
type ZxdgShellV6 struct {
	i uint32
	l ZxdgShellV6Listener
	c *Context
}

func newZxdgShellV6(c *Context) Object {
	o := &ZxdgShellV6{
		i: c.next(),
		c: c,
	}
	c.obj[o.i] = o
	return o
}

func init() {
	if constructors == nil {
		constructors = make(map[string]constructor)
	}
	constructors["zxdg_shell_v6"] = newZxdgShellV6
}

// ID returns the wayland object identifier
func (this *ZxdgShellV6) ID() uint32 {
	return this.i
}

// Type returns the string wayland type
func (this *ZxdgShellV6) Type() string {
	return "zxdg_shell_v6"
}

func (this *ZxdgShellV6) setListener(listener interface{}) error {
	l, ok := listener.(ZxdgShellV6Listener)
	if !ok {
		return errors.Errorf("listener must implement ZxdgShellV6 interface")
	}
	this.l = l
	return nil
}

func (this *ZxdgShellV6) dispatch(opCode uint16, payload []byte, file *os.File) {
	var len int
	_ = len
	switch opCode {
	case opCodeZxdgShellV6Ping:
		if this.l == nil {
			fmt.Println("ignoring Ping event: no listener")
		} else {
			fmt.Println("Received ZxdgShellV6 -> Ping: Dispatching")
			buf := bytes.NewBuffer(payload)
			_ = buf
			serial := hostByteOrder.Uint32(buf.Next(4))

			this.l.Ping(serial)
		}

	}
}

// Destroy this xdg_shell object.
//
// Destroying a bound xdg_shell object while there are surfaces
// still alive created by this xdg_shell object instance is illegal
// and will result in a protocol error.
func (this *ZxdgShellV6) Destroy() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgShellV6Destroy)

	fmt.Println("Sending ZxdgShellV6 -> Destroy")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Create a positioner object. A positioner object is used to position
// surfaces relative to some parent surface. See the interface description
// and xdg_surface.get_popup for details.
func (this *ZxdgShellV6) CreatePositioner(l ZxdgPositionerV6Listener) (*ZxdgPositionerV6, error) {
	if this == nil {
		return nil, errors.New("object is nil")
	}
	if this.c.Err != nil {
		return nil, errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return nil, errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := newZxdgPositionerV6(this.c).(*ZxdgPositionerV6)
	binary.Write(this.c.buf, hostByteOrder, uint32(ret.i))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgShellV6CreatePositioner)
	ret.l = l
	fmt.Println("Sending ZxdgShellV6 -> CreatePositioner")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return ret, nil
}

// This creates an xdg_surface for the given surface. While xdg_surface
// itself is not a role, the corresponding surface may only be assigned
// a role extending xdg_surface, such as xdg_toplevel or xdg_popup.
//
// This creates an xdg_surface for the given surface. An xdg_surface is
// used as basis to define a role to a given surface, such as xdg_toplevel
// or xdg_popup. It also manages functionality shared between xdg_surface
// based surface roles.
//
// See the documentation of xdg_surface for more details about what an
// xdg_surface is and how it is used.
func (this *ZxdgShellV6) GetXdgSurface(l ZxdgSurfaceV6Listener, surface uint32) (*ZxdgSurfaceV6, error) {
	if this == nil {
		return nil, errors.New("object is nil")
	}
	if this.c.Err != nil {
		return nil, errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return nil, errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := newZxdgSurfaceV6(this.c).(*ZxdgSurfaceV6)
	binary.Write(this.c.buf, hostByteOrder, uint32(ret.i))
	binary.Write(this.c.buf, hostByteOrder, uint32(surface))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgShellV6GetXdgSurface)
	ret.l = l
	fmt.Println("Sending ZxdgShellV6 -> GetXdgSurface")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return ret, nil
}

// A client must respond to a ping event with a pong request or
// the client may be deemed unresponsive. See xdg_shell.ping.
func (this *ZxdgShellV6) Pong(serial uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(serial))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgShellV6Pong)

	fmt.Println("Sending ZxdgShellV6 -> Pong")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

const (
	ZxdgPositionerV6ErrorInvalidInput = 0 // invalid input provided
)

const (
	ZxdgPositionerV6AnchorNone   = 0 // the center of the anchor rectangle
	ZxdgPositionerV6AnchorTop    = 1 // the top edge of the anchor rectangle
	ZxdgPositionerV6AnchorBottom = 2 // the bottom edge of the anchor rectangle
	ZxdgPositionerV6AnchorLeft   = 4 // the left edge of the anchor rectangle
	ZxdgPositionerV6AnchorRight  = 8 // the right edge of the anchor rectangle
)

const (
	ZxdgPositionerV6GravityNone   = 0 // center over the anchor edge
	ZxdgPositionerV6GravityTop    = 1 // position above the anchor edge
	ZxdgPositionerV6GravityBottom = 2 // position below the anchor edge
	ZxdgPositionerV6GravityLeft   = 4 // position to the left of the anchor edge
	ZxdgPositionerV6GravityRight  = 8 // position to the right of the anchor edge
)

const (
	ZxdgPositionerV6ConstraintAdjustmentNone    = 0  //
	ZxdgPositionerV6ConstraintAdjustmentSlideX  = 1  //
	ZxdgPositionerV6ConstraintAdjustmentSlideY  = 2  //
	ZxdgPositionerV6ConstraintAdjustmentFlipX   = 4  //
	ZxdgPositionerV6ConstraintAdjustmentFlipY   = 8  //
	ZxdgPositionerV6ConstraintAdjustmentResizeX = 16 //
	ZxdgPositionerV6ConstraintAdjustmentResizeY = 32 //
)

const (
	opCodeZxdgPositionerV6Destroy                 = 0
	opCodeZxdgPositionerV6SetSize                 = 1
	opCodeZxdgPositionerV6SetAnchorRect           = 2
	opCodeZxdgPositionerV6SetAnchor               = 3
	opCodeZxdgPositionerV6SetGravity              = 4
	opCodeZxdgPositionerV6SetConstraintAdjustment = 5
	opCodeZxdgPositionerV6SetOffset               = 6
)

// ZxdgPositionerV6 Events
type ZxdgPositionerV6Listener interface {
}

// The xdg_positioner provides a collection of rules for the placement of a
// child surface relative to a parent surface. Rules can be defined to ensure
// the child surface remains within the visible area's borders, and to
// specify how the child surface changes its position, such as sliding along
// an axis, or flipping around a rectangle. These positioner-created rules are
// constrained by the requirement that a child surface must intersect with or
// be at least partially adjacent to its parent surface.
//
// See the various requests for details about possible rules.
//
// At the time of the request, the compositor makes a copy of the rules
// specified by the xdg_positioner. Thus, after the request is complete the
// xdg_positioner object can be destroyed or reused; further changes to the
// object will have no effect on previous usages.
//
// For an xdg_positioner object to be considered complete, it must have a
// non-zero size set by set_size, and a non-zero anchor rectangle set by
// set_anchor_rect. Passing an incomplete xdg_positioner object when
// positioning a surface raises an error.
type ZxdgPositionerV6 struct {
	i uint32
	l ZxdgPositionerV6Listener
	c *Context
}

func newZxdgPositionerV6(c *Context) Object {
	o := &ZxdgPositionerV6{
		i: c.next(),
		c: c,
	}
	c.obj[o.i] = o
	return o
}

func init() {
	if constructors == nil {
		constructors = make(map[string]constructor)
	}
	constructors["zxdg_positioner_v6"] = newZxdgPositionerV6
}

// ID returns the wayland object identifier
func (this *ZxdgPositionerV6) ID() uint32 {
	return this.i
}

// Type returns the string wayland type
func (this *ZxdgPositionerV6) Type() string {
	return "zxdg_positioner_v6"
}

func (this *ZxdgPositionerV6) setListener(listener interface{}) error {
	l, ok := listener.(ZxdgPositionerV6Listener)
	if !ok {
		return errors.Errorf("listener must implement ZxdgPositionerV6 interface")
	}
	this.l = l
	return nil
}

func (this *ZxdgPositionerV6) dispatch(opCode uint16, payload []byte, file *os.File) {
	var len int
	_ = len
	switch opCode {

	}
}

// Notify the compositor that the xdg_positioner will no longer be used.
func (this *ZxdgPositionerV6) Destroy() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPositionerV6Destroy)

	fmt.Println("Sending ZxdgPositionerV6 -> Destroy")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Set the size of the surface that is to be positioned with the positioner
// object. The size is in surface-local coordinates and corresponds to the
// window geometry. See xdg_surface.set_window_geometry.
//
// If a zero or negative size is set the invalid_input error is raised.
func (this *ZxdgPositionerV6) SetSize(width int32, height int32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(width))
	binary.Write(this.c.buf, hostByteOrder, uint32(height))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPositionerV6SetSize)

	fmt.Println("Sending ZxdgPositionerV6 -> SetSize")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Specify the anchor rectangle within the parent surface that the child
// surface will be placed relative to. The rectangle is relative to the
// window geometry as defined by xdg_surface.set_window_geometry of the
// parent surface. The rectangle must be at least 1x1 large.
//
// When the xdg_positioner object is used to position a child surface, the
// anchor rectangle may not extend outside the window geometry of the
// positioned child's parent surface.
//
// If a zero or negative size is set the invalid_input error is raised.
func (this *ZxdgPositionerV6) SetAnchorRect(x int32, y int32, width int32, height int32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(x))
	binary.Write(this.c.buf, hostByteOrder, uint32(y))
	binary.Write(this.c.buf, hostByteOrder, uint32(width))
	binary.Write(this.c.buf, hostByteOrder, uint32(height))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPositionerV6SetAnchorRect)

	fmt.Println("Sending ZxdgPositionerV6 -> SetAnchorRect")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Defines a set of edges for the anchor rectangle. These are used to
// derive an anchor point that the child surface will be positioned
// relative to. If two orthogonal edges are specified (e.g. 'top' and
// 'left'), then the anchor point will be the intersection of the edges
// (e.g. the top left position of the rectangle); otherwise, the derived
// anchor point will be centered on the specified edge, or in the center of
// the anchor rectangle if no edge is specified.
//
// If two parallel anchor edges are specified (e.g. 'left' and 'right'),
// the invalid_input error is raised.
func (this *ZxdgPositionerV6) SetAnchor(anchor uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(anchor))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPositionerV6SetAnchor)

	fmt.Println("Sending ZxdgPositionerV6 -> SetAnchor")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Defines in what direction a surface should be positioned, relative to
// the anchor point of the parent surface. If two orthogonal gravities are
// specified (e.g. 'bottom' and 'right'), then the child surface will be
// placed in the specified direction; otherwise, the child surface will be
// centered over the anchor point on any axis that had no gravity
// specified.
//
// If two parallel gravities are specified (e.g. 'left' and 'right'), the
// invalid_input error is raised.
func (this *ZxdgPositionerV6) SetGravity(gravity uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(gravity))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPositionerV6SetGravity)

	fmt.Println("Sending ZxdgPositionerV6 -> SetGravity")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Specify how the window should be positioned if the originally intended
// position caused the surface to be constrained, meaning at least
// partially outside positioning boundaries set by the compositor. The
// adjustment is set by constructing a bitmask describing the adjustment to
// be made when the surface is constrained on that axis.
//
// If no bit for one axis is set, the compositor will assume that the child
// surface should not change its position on that axis when constrained.
//
// If more than one bit for one axis is set, the order of how adjustments
// are applied is specified in the corresponding adjustment descriptions.
//
// The default adjustment is none.
func (this *ZxdgPositionerV6) SetConstraintAdjustment(constraintAdjustment uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(constraintAdjustment))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPositionerV6SetConstraintAdjustment)

	fmt.Println("Sending ZxdgPositionerV6 -> SetConstraintAdjustment")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Specify the surface position offset relative to the position of the
// anchor on the anchor rectangle and the anchor on the surface. For
// example if the anchor of the anchor rectangle is at (x, y), the surface
// has the gravity bottom|right, and the offset is (ox, oy), the calculated
// surface position will be (x + ox, y + oy). The offset position of the
// surface is the one used for constraint testing. See
// set_constraint_adjustment.
//
// An example use case is placing a popup menu on top of a user interface
// element, while aligning the user interface element of the parent surface
// with some user interface element placed somewhere in the popup surface.
func (this *ZxdgPositionerV6) SetOffset(x int32, y int32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(x))
	binary.Write(this.c.buf, hostByteOrder, uint32(y))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPositionerV6SetOffset)

	fmt.Println("Sending ZxdgPositionerV6 -> SetOffset")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

const (
	ZxdgSurfaceV6ErrorNotConstructed     = 1 //
	ZxdgSurfaceV6ErrorAlreadyConstructed = 2 //
	ZxdgSurfaceV6ErrorUnconfiguredBuffer = 3 //
)

const (
	opCodeZxdgSurfaceV6Configure = 0
)

const (
	opCodeZxdgSurfaceV6Destroy           = 0
	opCodeZxdgSurfaceV6GetToplevel       = 1
	opCodeZxdgSurfaceV6GetPopup          = 2
	opCodeZxdgSurfaceV6SetWindowGeometry = 3
	opCodeZxdgSurfaceV6AckConfigure      = 4
)

// ZxdgSurfaceV6 Events
//
// Configure
// The configure event marks the end of a configure sequence. A configure
// sequence is a set of one or more events configuring the state of the
// xdg_surface, including the final xdg_surface.configure event.
//
// Where applicable, xdg_surface surface roles will during a configure
// sequence extend this event as a latched state sent as events before the
// xdg_surface.configure event. Such events should be considered to make up
// a set of atomically applied configuration states, where the
// xdg_surface.configure commits the accumulated state.
//
// Clients should arrange their surface for the new states, and then send
// an ack_configure request with the serial sent in this configure event at
// some point before committing the new surface.
//
// If the client receives multiple configure events before it can respond
// to one, it is free to discard all but the last event it received.
type ZxdgSurfaceV6Listener interface {
	Configure(serial uint32)
}

// An interface that may be implemented by a wl_surface, for
// implementations that provide a desktop-style user interface.
//
// It provides a base set of functionality required to construct user
// interface elements requiring management by the compositor, such as
// toplevel windows, menus, etc. The types of functionality are split into
// xdg_surface roles.
//
// Creating an xdg_surface does not set the role for a wl_surface. In order
// to map an xdg_surface, the client must create a role-specific object
// using, e.g., get_toplevel, get_popup. The wl_surface for any given
// xdg_surface can have at most one role, and may not be assigned any role
// not based on xdg_surface.
//
// A role must be assigned before any other requests are made to the
// xdg_surface object.
//
// The client must call wl_surface.commit on the corresponding wl_surface
// for the xdg_surface state to take effect.
//
// Creating an xdg_surface from a wl_surface which has a buffer attached or
// committed is a client error, and any attempts by a client to attach or
// manipulate a buffer prior to the first xdg_surface.configure call must
// also be treated as errors.
//
// For a surface to be mapped by the compositor, the following conditions
// must be met: (1) the client has assigned a xdg_surface based role to the
// surface, (2) the client has set and committed the xdg_surface state and
// the role dependent state to the surface and (3) the client has committed a
// buffer to the surface.
type ZxdgSurfaceV6 struct {
	i uint32
	l ZxdgSurfaceV6Listener
	c *Context
}

func newZxdgSurfaceV6(c *Context) Object {
	o := &ZxdgSurfaceV6{
		i: c.next(),
		c: c,
	}
	c.obj[o.i] = o
	return o
}

func init() {
	if constructors == nil {
		constructors = make(map[string]constructor)
	}
	constructors["zxdg_surface_v6"] = newZxdgSurfaceV6
}

// ID returns the wayland object identifier
func (this *ZxdgSurfaceV6) ID() uint32 {
	return this.i
}

// Type returns the string wayland type
func (this *ZxdgSurfaceV6) Type() string {
	return "zxdg_surface_v6"
}

func (this *ZxdgSurfaceV6) setListener(listener interface{}) error {
	l, ok := listener.(ZxdgSurfaceV6Listener)
	if !ok {
		return errors.Errorf("listener must implement ZxdgSurfaceV6 interface")
	}
	this.l = l
	return nil
}

func (this *ZxdgSurfaceV6) dispatch(opCode uint16, payload []byte, file *os.File) {
	var len int
	_ = len
	switch opCode {
	case opCodeZxdgSurfaceV6Configure:
		if this.l == nil {
			fmt.Println("ignoring Configure event: no listener")
		} else {
			fmt.Println("Received ZxdgSurfaceV6 -> Configure: Dispatching")
			buf := bytes.NewBuffer(payload)
			_ = buf
			serial := hostByteOrder.Uint32(buf.Next(4))

			this.l.Configure(serial)
		}

	}
}

// Destroy the xdg_surface object. An xdg_surface must only be destroyed
// after its role object has been destroyed.
func (this *ZxdgSurfaceV6) Destroy() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgSurfaceV6Destroy)

	fmt.Println("Sending ZxdgSurfaceV6 -> Destroy")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// This creates an xdg_toplevel object for the given xdg_surface and gives
// the associated wl_surface the xdg_toplevel role.
//
// See the documentation of xdg_toplevel for more details about what an
// xdg_toplevel is and how it is used.
func (this *ZxdgSurfaceV6) GetToplevel(l ZxdgToplevelV6Listener) (*ZxdgToplevelV6, error) {
	if this == nil {
		return nil, errors.New("object is nil")
	}
	if this.c.Err != nil {
		return nil, errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return nil, errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := newZxdgToplevelV6(this.c).(*ZxdgToplevelV6)
	binary.Write(this.c.buf, hostByteOrder, uint32(ret.i))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgSurfaceV6GetToplevel)
	ret.l = l
	fmt.Println("Sending ZxdgSurfaceV6 -> GetToplevel")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return ret, nil
}

// This creates an xdg_popup object for the given xdg_surface and gives the
// associated wl_surface the xdg_popup role.
//
// See the documentation of xdg_popup for more details about what an
// xdg_popup is and how it is used.
func (this *ZxdgSurfaceV6) GetPopup(l ZxdgPopupV6Listener, parent uint32, positioner uint32) (*ZxdgPopupV6, error) {
	if this == nil {
		return nil, errors.New("object is nil")
	}
	if this.c.Err != nil {
		return nil, errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return nil, errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	ret := newZxdgPopupV6(this.c).(*ZxdgPopupV6)
	binary.Write(this.c.buf, hostByteOrder, uint32(ret.i))
	binary.Write(this.c.buf, hostByteOrder, uint32(parent))
	binary.Write(this.c.buf, hostByteOrder, uint32(positioner))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgSurfaceV6GetPopup)
	ret.l = l
	fmt.Println("Sending ZxdgSurfaceV6 -> GetPopup")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return ret, nil
}

// The window geometry of a surface is its "visible bounds" from the
// user's perspective. Client-side decorations often have invisible
// portions like drop-shadows which should be ignored for the
// purposes of aligning, placing and constraining windows.
//
// The window geometry is double buffered, and will be applied at the
// time wl_surface.commit of the corresponding wl_surface is called.
//
// Once the window geometry of the surface is set, it is not possible to
// unset it, and it will remain the same until set_window_geometry is
// called again, even if a new subsurface or buffer is attached.
//
// If never set, the value is the full bounds of the surface,
// including any subsurfaces. This updates dynamically on every
// commit. This unset is meant for extremely simple clients.
//
// The arguments are given in the surface-local coordinate space of
// the wl_surface associated with this xdg_surface.
//
// The width and height must be greater than zero. Setting an invalid size
// will raise an error. When applied, the effective window geometry will be
// the set window geometry clamped to the bounding rectangle of the
// combined geometry of the surface of the xdg_surface and the associated
// subsurfaces.
func (this *ZxdgSurfaceV6) SetWindowGeometry(x int32, y int32, width int32, height int32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(x))
	binary.Write(this.c.buf, hostByteOrder, uint32(y))
	binary.Write(this.c.buf, hostByteOrder, uint32(width))
	binary.Write(this.c.buf, hostByteOrder, uint32(height))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgSurfaceV6SetWindowGeometry)

	fmt.Println("Sending ZxdgSurfaceV6 -> SetWindowGeometry")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// When a configure event is received, if a client commits the
// surface in response to the configure event, then the client
// must make an ack_configure request sometime before the commit
// request, passing along the serial of the configure event.
//
// For instance, for toplevel surfaces the compositor might use this
// information to move a surface to the top left only when the client has
// drawn itself for the maximized or fullscreen state.
//
// If the client receives multiple configure events before it
// can respond to one, it only has to ack the last configure event.
//
// A client is not required to commit immediately after sending
// an ack_configure request - it may even ack_configure several times
// before its next surface commit.
//
// A client may send multiple ack_configure requests before committing, but
// only the last request sent before a commit indicates which configure
// event the client really is responding to.
func (this *ZxdgSurfaceV6) AckConfigure(serial uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(serial))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgSurfaceV6AckConfigure)

	fmt.Println("Sending ZxdgSurfaceV6 -> AckConfigure")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

const (
	ZxdgToplevelV6ResizeEdgeNone        = 0  //
	ZxdgToplevelV6ResizeEdgeTop         = 1  //
	ZxdgToplevelV6ResizeEdgeBottom      = 2  //
	ZxdgToplevelV6ResizeEdgeLeft        = 4  //
	ZxdgToplevelV6ResizeEdgeTopLeft     = 5  //
	ZxdgToplevelV6ResizeEdgeBottomLeft  = 6  //
	ZxdgToplevelV6ResizeEdgeRight       = 8  //
	ZxdgToplevelV6ResizeEdgeTopRight    = 9  //
	ZxdgToplevelV6ResizeEdgeBottomRight = 10 //
)

const (
	ZxdgToplevelV6StateMaximized  = 1 // the surface is maximized
	ZxdgToplevelV6StateFullscreen = 2 // the surface is fullscreen
	ZxdgToplevelV6StateResizing   = 3 // the surface is being resized
	ZxdgToplevelV6StateActivated  = 4 // the surface is now activated
)

const (
	opCodeZxdgToplevelV6Configure = 0
	opCodeZxdgToplevelV6Close     = 1
)

const (
	opCodeZxdgToplevelV6Destroy         = 0
	opCodeZxdgToplevelV6SetParent       = 1
	opCodeZxdgToplevelV6SetTitle        = 2
	opCodeZxdgToplevelV6SetAppID        = 3
	opCodeZxdgToplevelV6ShowWindowMenu  = 4
	opCodeZxdgToplevelV6Move            = 5
	opCodeZxdgToplevelV6Resize          = 6
	opCodeZxdgToplevelV6SetMaxSize      = 7
	opCodeZxdgToplevelV6SetMinSize      = 8
	opCodeZxdgToplevelV6SetMaximized    = 9
	opCodeZxdgToplevelV6UnsetMaximized  = 10
	opCodeZxdgToplevelV6SetFullscreen   = 11
	opCodeZxdgToplevelV6UnsetFullscreen = 12
	opCodeZxdgToplevelV6SetMinimized    = 13
)

// ZxdgToplevelV6 Events
//
// Configure
// This configure event asks the client to resize its toplevel surface or
// to change its state. The configured state should not be applied
// immediately. See xdg_surface.configure for details.
//
// The width and height arguments specify a hint to the window
// about how its surface should be resized in window geometry
// coordinates. See set_window_geometry.
//
// If the width or height arguments are zero, it means the client
// should decide its own window dimension. This may happen when the
// compositor needs to configure the state of the surface but doesn't
// have any information about any previous or expected dimension.
//
// The states listed in the event specify how the width/height
// arguments should be interpreted, and possibly how it should be
// drawn.
//
// Clients must send an ack_configure in response to this event. See
// xdg_surface.configure and xdg_surface.ack_configure for details.
//
// Close
// The close event is sent by the compositor when the user
// wants the surface to be closed. This should be equivalent to
// the user clicking the close button in client-side decorations,
// if your application has any.
//
// This is only a request that the user intends to close the
// window. The client may choose to ignore this request, or show
// a dialog to ask the user to save their data, etc.
type ZxdgToplevelV6Listener interface {
	Configure(width int32, height int32, states []byte)
	Close()
}

// This interface defines an xdg_surface role which allows a surface to,
// among other things, set window-like properties such as maximize,
// fullscreen, and minimize, set application-specific metadata like title and
// id, and well as trigger user interactive operations such as interactive
// resize and move.
type ZxdgToplevelV6 struct {
	i uint32
	l ZxdgToplevelV6Listener
	c *Context
}

func newZxdgToplevelV6(c *Context) Object {
	o := &ZxdgToplevelV6{
		i: c.next(),
		c: c,
	}
	c.obj[o.i] = o
	return o
}

func init() {
	if constructors == nil {
		constructors = make(map[string]constructor)
	}
	constructors["zxdg_toplevel_v6"] = newZxdgToplevelV6
}

// ID returns the wayland object identifier
func (this *ZxdgToplevelV6) ID() uint32 {
	return this.i
}

// Type returns the string wayland type
func (this *ZxdgToplevelV6) Type() string {
	return "zxdg_toplevel_v6"
}

func (this *ZxdgToplevelV6) setListener(listener interface{}) error {
	l, ok := listener.(ZxdgToplevelV6Listener)
	if !ok {
		return errors.Errorf("listener must implement ZxdgToplevelV6 interface")
	}
	this.l = l
	return nil
}

func (this *ZxdgToplevelV6) dispatch(opCode uint16, payload []byte, file *os.File) {
	var len int
	_ = len
	switch opCode {
	case opCodeZxdgToplevelV6Configure:
		if this.l == nil {
			fmt.Println("ignoring Configure event: no listener")
		} else {
			fmt.Println("Received ZxdgToplevelV6 -> Configure: Dispatching")
			buf := bytes.NewBuffer(payload)
			_ = buf
			width := int32(hostByteOrder.Uint32(buf.Next(4)))
			height := int32(hostByteOrder.Uint32(buf.Next(4)))
			len = int(hostByteOrder.Uint32(buf.Next(4)))
			states := make([]byte, len)
			buf.Read(states)
			if len%4 != 0 {
				buf.Next(4 - (len % 4))
			}

			this.l.Configure(width, height, states)
		}
	case opCodeZxdgToplevelV6Close:
		if this.l == nil {
			fmt.Println("ignoring Close event: no listener")
		} else {
			fmt.Println("Received ZxdgToplevelV6 -> Close: Dispatching")

			this.l.Close()
		}

	}
}

// Unmap and destroy the window. The window will be effectively
// hidden from the user's point of view, and all state like
// maximization, fullscreen, and so on, will be lost.
func (this *ZxdgToplevelV6) Destroy() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6Destroy)

	fmt.Println("Sending ZxdgToplevelV6 -> Destroy")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Set the "parent" of this surface. This window should be stacked
// above a parent. The parent surface must be mapped as long as this
// surface is mapped.
//
// Parent windows should be set on dialogs, toolboxes, or other
// "auxiliary" surfaces, so that the parent is raised when the dialog
// is raised.
func (this *ZxdgToplevelV6) SetParent(parent uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(parent))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6SetParent)

	fmt.Println("Sending ZxdgToplevelV6 -> SetParent")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Set a short title for the surface.
//
// This string may be used to identify the surface in a task bar,
// window list, or other user interface elements provided by the
// compositor.
//
// The string must be encoded in UTF-8.
func (this *ZxdgToplevelV6) SetTitle(title string) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(len(title)+1))
	this.c.buf.WriteString(title)
	this.c.buf.WriteByte(0)
	if (len(title)+1)%4 != 0 {
		this.c.buf.Write(make([]byte, 4-(len(title)+1)%4))
	}
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6SetTitle)

	fmt.Println("Sending ZxdgToplevelV6 -> SetTitle")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Set an application identifier for the surface.
//
// The app ID identifies the general class of applications to which
// the surface belongs. The compositor can use this to group multiple
// surfaces together, or to determine how to launch a new application.
//
// For D-Bus activatable applications, the app ID is used as the D-Bus
// service name.
//
// The compositor shell will try to group application surfaces together
// by their app ID. As a best practice, it is suggested to select app
// ID's that match the basename of the application's .desktop file.
// For example, "org.freedesktop.FooViewer" where the .desktop file is
// "org.freedesktop.FooViewer.desktop".
//
// See the desktop-entry specification [0] for more details on
// application identifiers and how they relate to well-known D-Bus
// names and .desktop files.
//
// [0] http://standards.freedesktop.org/desktop-entry-spec/
func (this *ZxdgToplevelV6) SetAppID(appID string) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(len(appID)+1))
	this.c.buf.WriteString(appID)
	this.c.buf.WriteByte(0)
	if (len(appID)+1)%4 != 0 {
		this.c.buf.Write(make([]byte, 4-(len(appID)+1)%4))
	}
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6SetAppID)

	fmt.Println("Sending ZxdgToplevelV6 -> SetAppID")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Clients implementing client-side decorations might want to show
// a context menu when right-clicking on the decorations, giving the
// user a menu that they can use to maximize or minimize the window.
//
// This request asks the compositor to pop up such a window menu at
// the given position, relative to the local surface coordinates of
// the parent surface. There are no guarantees as to what menu items
// the window menu contains.
//
// This request must be used in response to some sort of user action
// like a button press, key press, or touch down event.
func (this *ZxdgToplevelV6) ShowWindowMenu(seat uint32, serial uint32, x int32, y int32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(seat))
	binary.Write(this.c.buf, hostByteOrder, uint32(serial))
	binary.Write(this.c.buf, hostByteOrder, uint32(x))
	binary.Write(this.c.buf, hostByteOrder, uint32(y))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6ShowWindowMenu)

	fmt.Println("Sending ZxdgToplevelV6 -> ShowWindowMenu")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Start an interactive, user-driven move of the surface.
//
// This request must be used in response to some sort of user action
// like a button press, key press, or touch down event. The passed
// serial is used to determine the type of interactive move (touch,
// pointer, etc).
//
// The server may ignore move requests depending on the state of
// the surface (e.g. fullscreen or maximized), or if the passed serial
// is no longer valid.
//
// If triggered, the surface will lose the focus of the device
// (wl_pointer, wl_touch, etc) used for the move. It is up to the
// compositor to visually indicate that the move is taking place, such as
// updating a pointer cursor, during the move. There is no guarantee
// that the device focus will return when the move is completed.
func (this *ZxdgToplevelV6) Move(seat uint32, serial uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(seat))
	binary.Write(this.c.buf, hostByteOrder, uint32(serial))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6Move)

	fmt.Println("Sending ZxdgToplevelV6 -> Move")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Start a user-driven, interactive resize of the surface.
//
// This request must be used in response to some sort of user action
// like a button press, key press, or touch down event. The passed
// serial is used to determine the type of interactive resize (touch,
// pointer, etc).
//
// The server may ignore resize requests depending on the state of
// the surface (e.g. fullscreen or maximized).
//
// If triggered, the client will receive configure events with the
// "resize" state enum value and the expected sizes. See the "resize"
// enum value for more details about what is required. The client
// must also acknowledge configure events using "ack_configure". After
// the resize is completed, the client will receive another "configure"
// event without the resize state.
//
// If triggered, the surface also will lose the focus of the device
// (wl_pointer, wl_touch, etc) used for the resize. It is up to the
// compositor to visually indicate that the resize is taking place,
// such as updating a pointer cursor, during the resize. There is no
// guarantee that the device focus will return when the resize is
// completed.
//
// The edges parameter specifies how the surface should be resized,
// and is one of the values of the resize_edge enum. The compositor
// may use this information to update the surface position for
// example when dragging the top left corner. The compositor may also
// use this information to adapt its behavior, e.g. choose an
// appropriate cursor image.
func (this *ZxdgToplevelV6) Resize(seat uint32, serial uint32, edges uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(seat))
	binary.Write(this.c.buf, hostByteOrder, uint32(serial))
	binary.Write(this.c.buf, hostByteOrder, uint32(edges))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6Resize)

	fmt.Println("Sending ZxdgToplevelV6 -> Resize")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Set a maximum size for the window.
//
// The client can specify a maximum size so that the compositor does
// not try to configure the window beyond this size.
//
// The width and height arguments are in window geometry coordinates.
// See xdg_surface.set_window_geometry.
//
// Values set in this way are double-buffered. They will get applied
// on the next commit.
//
// The compositor can use this information to allow or disallow
// different states like maximize or fullscreen and draw accurate
// animations.
//
// Similarly, a tiling window manager may use this information to
// place and resize client windows in a more effective way.
//
// The client should not rely on the compositor to obey the maximum
// size. The compositor may decide to ignore the values set by the
// client and request a larger size.
//
// If never set, or a value of zero in the request, means that the
// client has no expected maximum size in the given dimension.
// As a result, a client wishing to reset the maximum size
// to an unspecified state can use zero for width and height in the
// request.
//
// Requesting a maximum size to be smaller than the minimum size of
// a surface is illegal and will result in a protocol error.
//
// The width and height must be greater than or equal to zero. Using
// strictly negative values for width and height will result in a
// protocol error.
func (this *ZxdgToplevelV6) SetMaxSize(width int32, height int32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(width))
	binary.Write(this.c.buf, hostByteOrder, uint32(height))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6SetMaxSize)

	fmt.Println("Sending ZxdgToplevelV6 -> SetMaxSize")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Set a minimum size for the window.
//
// The client can specify a minimum size so that the compositor does
// not try to configure the window below this size.
//
// The width and height arguments are in window geometry coordinates.
// See xdg_surface.set_window_geometry.
//
// Values set in this way are double-buffered. They will get applied
// on the next commit.
//
// The compositor can use this information to allow or disallow
// different states like maximize or fullscreen and draw accurate
// animations.
//
// Similarly, a tiling window manager may use this information to
// place and resize client windows in a more effective way.
//
// The client should not rely on the compositor to obey the minimum
// size. The compositor may decide to ignore the values set by the
// client and request a smaller size.
//
// If never set, or a value of zero in the request, means that the
// client has no expected minimum size in the given dimension.
// As a result, a client wishing to reset the minimum size
// to an unspecified state can use zero for width and height in the
// request.
//
// Requesting a minimum size to be larger than the maximum size of
// a surface is illegal and will result in a protocol error.
//
// The width and height must be greater than or equal to zero. Using
// strictly negative values for width and height will result in a
// protocol error.
func (this *ZxdgToplevelV6) SetMinSize(width int32, height int32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(width))
	binary.Write(this.c.buf, hostByteOrder, uint32(height))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6SetMinSize)

	fmt.Println("Sending ZxdgToplevelV6 -> SetMinSize")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Maximize the surface.
//
// After requesting that the surface should be maximized, the compositor
// will respond by emitting a configure event with the "maximized" state
// and the required window geometry. The client should then update its
// content, drawing it in a maximized state, i.e. without shadow or other
// decoration outside of the window geometry. The client must also
// acknowledge the configure when committing the new content (see
// ack_configure).
//
// It is up to the compositor to decide how and where to maximize the
// surface, for example which output and what region of the screen should
// be used.
//
// If the surface was already maximized, the compositor will still emit
// a configure event with the "maximized" state.
func (this *ZxdgToplevelV6) SetMaximized() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6SetMaximized)

	fmt.Println("Sending ZxdgToplevelV6 -> SetMaximized")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Unmaximize the surface.
//
// After requesting that the surface should be unmaximized, the compositor
// will respond by emitting a configure event without the "maximized"
// state. If available, the compositor will include the window geometry
// dimensions the window had prior to being maximized in the configure
// request. The client must then update its content, drawing it in a
// regular state, i.e. potentially with shadow, etc. The client must also
// acknowledge the configure when committing the new content (see
// ack_configure).
//
// It is up to the compositor to position the surface after it was
// unmaximized; usually the position the surface had before maximizing, if
// applicable.
//
// If the surface was already not maximized, the compositor will still
// emit a configure event without the "maximized" state.
func (this *ZxdgToplevelV6) UnsetMaximized() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6UnsetMaximized)

	fmt.Println("Sending ZxdgToplevelV6 -> UnsetMaximized")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Make the surface fullscreen.
//
// You can specify an output that you would prefer to be fullscreen.
// If this value is NULL, it's up to the compositor to choose which
// display will be used to map this surface.
//
// If the surface doesn't cover the whole output, the compositor will
// position the surface in the center of the output and compensate with
// black borders filling the rest of the output.
func (this *ZxdgToplevelV6) SetFullscreen(output uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(output))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6SetFullscreen)

	fmt.Println("Sending ZxdgToplevelV6 -> SetFullscreen")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

func (this *ZxdgToplevelV6) UnsetFullscreen() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6UnsetFullscreen)

	fmt.Println("Sending ZxdgToplevelV6 -> UnsetFullscreen")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// Request that the compositor minimize your surface. There is no
// way to know if the surface is currently minimized, nor is there
// any way to unset minimization on this surface.
//
// If you are looking to throttle redrawing when minimized, please
// instead use the wl_surface.frame event for this, as this will
// also work with live previews on windows in Alt-Tab, Expose or
// similar compositor features.
func (this *ZxdgToplevelV6) SetMinimized() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgToplevelV6SetMinimized)

	fmt.Println("Sending ZxdgToplevelV6 -> SetMinimized")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

const (
	ZxdgPopupV6ErrorInvalidGrab = 0 // tried to grab after being mapped
)

const (
	opCodeZxdgPopupV6Configure = 0
	opCodeZxdgPopupV6PopupDone = 1
)

const (
	opCodeZxdgPopupV6Destroy = 0
	opCodeZxdgPopupV6Grab    = 1
)

// ZxdgPopupV6 Events
//
// Configure
// This event asks the popup surface to configure itself given the
// configuration. The configured state should not be applied immediately.
// See xdg_surface.configure for details.
//
// The x and y arguments represent the position the popup was placed at
// given the xdg_positioner rule, relative to the upper left corner of the
// window geometry of the parent surface.
//
// PopupDone
// The popup_done event is sent out when a popup is dismissed by the
// compositor. The client should destroy the xdg_popup object at this
// point.
type ZxdgPopupV6Listener interface {
	Configure(x int32, y int32, width int32, height int32)
	PopupDone()
}

// A popup surface is a short-lived, temporary surface. It can be used to
// implement for example menus, popovers, tooltips and other similar user
// interface concepts.
//
// A popup can be made to take an explicit grab. See xdg_popup.grab for
// details.
//
// When the popup is dismissed, a popup_done event will be sent out, and at
// the same time the surface will be unmapped. See the xdg_popup.popup_done
// event for details.
//
// Explicitly destroying the xdg_popup object will also dismiss the popup and
// unmap the surface. Clients that want to dismiss the popup when another
// surface of their own is clicked should dismiss the popup using the destroy
// request.
//
// The parent surface must have either the xdg_toplevel or xdg_popup surface
// role.
//
// A newly created xdg_popup will be stacked on top of all previously created
// xdg_popup surfaces associated with the same xdg_toplevel.
//
// The parent of an xdg_popup must be mapped (see the xdg_surface
// description) before the xdg_popup itself.
//
// The x and y arguments passed when creating the popup object specify
// where the top left of the popup should be placed, relative to the
// local surface coordinates of the parent surface. See
// xdg_surface.get_popup. An xdg_popup must intersect with or be at least
// partially adjacent to its parent surface.
//
// The client must call wl_surface.commit on the corresponding wl_surface
// for the xdg_popup state to take effect.
type ZxdgPopupV6 struct {
	i uint32
	l ZxdgPopupV6Listener
	c *Context
}

func newZxdgPopupV6(c *Context) Object {
	o := &ZxdgPopupV6{
		i: c.next(),
		c: c,
	}
	c.obj[o.i] = o
	return o
}

func init() {
	if constructors == nil {
		constructors = make(map[string]constructor)
	}
	constructors["zxdg_popup_v6"] = newZxdgPopupV6
}

// ID returns the wayland object identifier
func (this *ZxdgPopupV6) ID() uint32 {
	return this.i
}

// Type returns the string wayland type
func (this *ZxdgPopupV6) Type() string {
	return "zxdg_popup_v6"
}

func (this *ZxdgPopupV6) setListener(listener interface{}) error {
	l, ok := listener.(ZxdgPopupV6Listener)
	if !ok {
		return errors.Errorf("listener must implement ZxdgPopupV6 interface")
	}
	this.l = l
	return nil
}

func (this *ZxdgPopupV6) dispatch(opCode uint16, payload []byte, file *os.File) {
	var len int
	_ = len
	switch opCode {
	case opCodeZxdgPopupV6Configure:
		if this.l == nil {
			fmt.Println("ignoring Configure event: no listener")
		} else {
			fmt.Println("Received ZxdgPopupV6 -> Configure: Dispatching")
			buf := bytes.NewBuffer(payload)
			_ = buf
			x := int32(hostByteOrder.Uint32(buf.Next(4)))
			y := int32(hostByteOrder.Uint32(buf.Next(4)))
			width := int32(hostByteOrder.Uint32(buf.Next(4)))
			height := int32(hostByteOrder.Uint32(buf.Next(4)))

			this.l.Configure(x, y, width, height)
		}
	case opCodeZxdgPopupV6PopupDone:
		if this.l == nil {
			fmt.Println("ignoring PopupDone event: no listener")
		} else {
			fmt.Println("Received ZxdgPopupV6 -> PopupDone: Dispatching")

			this.l.PopupDone()
		}

	}
}

// This destroys the popup. Explicitly destroying the xdg_popup
// object will also dismiss the popup, and unmap the surface.
//
// If this xdg_popup is not the "topmost" popup, a protocol error
// will be sent.
func (this *ZxdgPopupV6) Destroy() error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPopupV6Destroy)

	fmt.Println("Sending ZxdgPopupV6 -> Destroy")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}

// This request makes the created popup take an explicit grab. An explicit
// grab will be dismissed when the user dismisses the popup, or when the
// client destroys the xdg_popup. This can be done by the user clicking
// outside the surface, using the keyboard, or even locking the screen
// through closing the lid or a timeout.
//
// If the compositor denies the grab, the popup will be immediately
// dismissed.
//
// This request must be used in response to some sort of user action like a
// button press, key press, or touch down event. The serial number of the
// event should be passed as 'serial'.
//
// The parent of a grabbing popup must either be an xdg_toplevel surface or
// another xdg_popup with an explicit grab. If the parent is another
// xdg_popup it means that the popups are nested, with this popup now being
// the topmost popup.
//
// Nested popups must be destroyed in the reverse order they were created
// in, e.g. the only popup you are allowed to destroy at all times is the
// topmost one.
//
// When compositors choose to dismiss a popup, they may dismiss every
// nested grabbing popup as well. When a compositor dismisses popups, it
// will follow the same dismissing order as required from the client.
//
// The parent of a grabbing popup must either be another xdg_popup with an
// active explicit grab, or an xdg_popup or xdg_toplevel, if there are no
// explicit grabs already taken.
//
// If the topmost grabbing popup is destroyed, the grab will be returned to
// the parent of the popup, if that parent previously had an explicit grab.
//
// If the parent is a grabbing popup which has already been dismissed, this
// popup will be immediately dismissed. If the parent is a popup that did
// not take an explicit grab, an error will be raised.
//
// During a popup grab, the client owning the grab will receive pointer
// and touch events for all their surfaces as normal (similar to an
// "owner-events" grab in X11 parlance), while the top most grabbing popup
// will always have keyboard focus.
func (this *ZxdgPopupV6) Grab(seat uint32, serial uint32) error {
	if this == nil {
		return errors.New("object is nil")
	}
	if this.c.Err != nil {
		return errors.Wrap(this.c.Err, "global wayland error")
	}
	this.c.mu.Lock()
	defer this.c.mu.Unlock()
	_, exists := this.c.obj[this.i]
	if !exists {
		return errors.New("object has been deleted")
	}
	this.c.buf.Reset()
	var tmp int32
	_ = tmp
	var oob []byte
	binary.Write(this.c.buf, hostByteOrder, this.i)
	binary.Write(this.c.buf, hostByteOrder, uint32(0))
	binary.Write(this.c.buf, hostByteOrder, uint32(seat))
	binary.Write(this.c.buf, hostByteOrder, uint32(serial))
	hostByteOrder.PutUint32(this.c.buf.Bytes()[4:8], uint32(this.c.buf.Len())<<16|opCodeZxdgPopupV6Grab)

	fmt.Println("Sending ZxdgPopupV6 -> Grab")
	fmt.Println(hex.Dump(this.c.buf.Bytes()))
	this.c.c.WriteMsgUnix(this.c.buf.Bytes(), oob, nil)
	return nil
}
