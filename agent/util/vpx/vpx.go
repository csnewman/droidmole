package vpx

import (
	"errors"
	"log"
	"unsafe"
)

/*
#cgo pkg-config: vpx
#include <vpx/vpx_codec.h>
#include <vpx/vpx_encoder.h>
#include <vpx/vpx_decoder.h>
#include <vpx/vp8.h>
#include <vpx/vp8cx.h>
#include <stdlib.h>
#include <string.h>

typedef struct BufferWrapper {
    void *ptr;
    int size;
} BufferWrapperType;

vpx_codec_err_t vpx_codec_enc_init_cust(vpx_codec_ctx_t *ctx,
                                       vpx_codec_iface_t *iface,
                                       const vpx_codec_enc_cfg_t *cfg,
                                       vpx_codec_flags_t flags) {
    return vpx_codec_enc_init_ver(ctx, iface, cfg, flags, VPX_ENCODER_ABI_VERSION);
}

int vpx_img_plane_width(const vpx_image_t *img, int plane) {
    if (plane > 0 && img->x_chroma_shift > 0)
        return (img->d_w + 1) >> img->x_chroma_shift;
    else
        return img->d_w;
}

int vpx_img_plane_height(const vpx_image_t *img, int plane) {
    if (plane > 0 && img->y_chroma_shift > 0)
        return (img->d_h + 1) >> img->y_chroma_shift;
    else
        return img->d_h;
}

int vpx_img_read(vpx_image_t *img, void *blob) {
    int plane;

    for (plane = 0; plane < 3; ++plane) {
        unsigned char *buf = img->planes[plane];
        const int stride = img->stride[plane];
        const int w = vpx_img_plane_width(img, plane) * ((img->fmt & VPX_IMG_FMT_HIGHBITDEPTH) ? 2 : 1);
        const int h = vpx_img_plane_height(img, plane);
        int y;

        for (y = 0; y < h; ++y) {
            memcpy(buf, blob, w);
            buf += stride;
            blob += w;
        }
    }
}

BufferWrapperType vpx_codec_get_frame_buffer(vpx_codec_ctx_t *codec, vpx_codec_iter_t *iter) {
    BufferWrapperType buffer = {NULL, 0};
    const vpx_codec_cx_pkt_t *pkt = vpx_codec_get_cx_data(codec, iter);
    if (pkt != NULL && pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
        buffer.ptr = pkt->data.frame.buf;
        buffer.size = pkt->data.frame.sz;
    }
    return buffer;
}

*/
import "C"

type CodecCtx C.vpx_codec_ctx_t
type CodecEncCfg C.vpx_codec_enc_cfg_t
type CodecIface C.vpx_codec_iface_t
type CodecPts C.vpx_codec_pts_t
type EncFrameFlags C.vpx_enc_frame_flags_t
type CodecIter C.vpx_codec_iter_t

const codecCtxSize = unsafe.Sizeof([1]C.vpx_codec_ctx_t{})
const codecEncCfgSize = unsafe.Sizeof([1]C.vpx_codec_enc_cfg_t{})

const (
	EFlagNone    EncFrameFlags = 0
	EFlagForceKF EncFrameFlags = 1

	DLRealtime    uint64 = 1
	DLGoodQuality uint64 = 1000000
	DLBestQuality uint64 = 0
)

var (
	UnknownError         = errors.New("unknown error")
	MemoryError          = errors.New("memory error")
	ABIMismatch          = errors.New("ABI mismatch")
	Incapable            = errors.New("incapable")
	UnsupportedBitstream = errors.New("unsupported bitstream")
	UnsupportedFeature   = errors.New("unsupported feature")
	CorruptFrame         = errors.New("corrupt frame")
	InvalidParam         = errors.New("invalid param")
)

func allocMemory(size uintptr) unsafe.Pointer {
	ptr, err := C.calloc(C.size_t(1), (C.size_t)(size))
	if err != nil {
		log.Fatal("allocation error", err.Error())
	}

	return ptr
}

func NewCodecCtx() *CodecCtx {
	return (*CodecCtx)(allocMemory(codecCtxSize))
}

func (c *CodecCtx) EncInit(iface *CodecIface, cfg *CodecEncCfg, flags int) error {
	return convertCodecError(C.vpx_codec_enc_init_cust(
		(*C.vpx_codec_ctx_t)(c),
		(*C.vpx_codec_iface_t)(iface),
		(*C.vpx_codec_enc_cfg_t)(cfg),
		(C.vpx_codec_flags_t)(flags),
	))
}

func (c *CodecCtx) Encode(img *Image, pts CodecPts, duration uint64, flags EncFrameFlags, deadline uint64) error {
	return convertCodecError(C.vpx_codec_encode(
		(*C.vpx_codec_ctx_t)(c),
		(*C.vpx_image_t)(img),
		(C.vpx_codec_pts_t)(pts),
		(C.ulong)(duration),
		(C.vpx_enc_frame_flags_t)(flags),
		(C.ulong)(deadline),
	))
}

func (c *CodecCtx) GetFrameBuffer(iter *CodecIter) []byte {
	buffer := C.vpx_codec_get_frame_buffer(
		(*C.vpx_codec_ctx_t)(c),
		(*C.vpx_codec_iter_t)(iter),
	)
	if buffer.ptr == nil {
		return nil
	}
	return C.GoBytes(buffer.ptr, buffer.size)
}

func (c *CodecCtx) Free() {
	C.free(unsafe.Pointer(c))
}

func NewCodecEncCfg() *CodecEncCfg {
	return (*CodecEncCfg)(allocMemory(codecEncCfgSize))
}

func (c *CodecEncCfg) Default(iface *CodecIface) error {
	return convertCodecError(C.vpx_codec_enc_config_default(
		(*C.vpx_codec_iface_t)(iface),
		(*C.vpx_codec_enc_cfg_t)(c),
		0,
	))
}

func (c *CodecEncCfg) SetGW(width uint) {
	c.g_w = C.uint(width)
}

func (c *CodecEncCfg) SetGH(height uint) {
	c.g_h = C.uint(height)
}

func (c *CodecEncCfg) SetRcTargetBitrate(rate uint) {
	c.rc_target_bitrate = C.uint(rate)
}

func (c *CodecEncCfg) SetGErrorResilient(res uint) {
	c.g_error_resilient = C.uint(res)
}

func (c *CodecEncCfg) SetGTimebase(num int, den int) {
	c.g_timebase.num = C.int(num)
	c.g_timebase.den = C.int(den)
}

func (c *CodecEncCfg) Free() {
	C.free(unsafe.Pointer(c))
}

func VP8Iface() *CodecIface {
	return (*CodecIface)(C.vpx_codec_vp8_cx())
}

func convertCodecError(err C.vpx_codec_err_t) error {
	switch err {
	case C.VPX_CODEC_OK:
		return nil
	case C.VPX_CODEC_ERROR:
		return UnknownError
	case C.VPX_CODEC_MEM_ERROR:
		return MemoryError
	case C.VPX_CODEC_ABI_MISMATCH:
		return ABIMismatch
	case C.VPX_CODEC_INCAPABLE:
		return Incapable
	case C.VPX_CODEC_UNSUP_BITSTREAM:
		return UnsupportedBitstream
	case C.VPX_CODEC_UNSUP_FEATURE:
		return UnsupportedFeature
	case C.VPX_CODEC_CORRUPT_FRAME:
		return CorruptFrame
	case C.VPX_CODEC_INVALID_PARAM:
		return InvalidParam
	default:
		return UnknownError
	}
}

type Image C.vpx_image_t
type ImageFormat C.vpx_img_fmt_t

const (
	ImageFormatNone   ImageFormat = C.VPX_IMG_FMT_NONE
	ImageFormatYv12   ImageFormat = C.VPX_IMG_FMT_YV12
	ImageFormatI420   ImageFormat = C.VPX_IMG_FMT_I420
	ImageFormatI422   ImageFormat = C.VPX_IMG_FMT_I422
	ImageFormatI444   ImageFormat = C.VPX_IMG_FMT_I444
	ImageFormatI440   ImageFormat = C.VPX_IMG_FMT_I440
	ImageFormatI42016 ImageFormat = C.VPX_IMG_FMT_I42016
	ImageFormatI42216 ImageFormat = C.VPX_IMG_FMT_I42216
	ImageFormatI44416 ImageFormat = C.VPX_IMG_FMT_I44416
	ImageFormatI44016 ImageFormat = C.VPX_IMG_FMT_I44016
)

func NullImage() *Image {
	return nil
}

func (i *Image) Alloc(fmt ImageFormat, dW uint32, dH uint32, align uint32) *Image {
	return (*Image)(C.vpx_img_alloc(
		(*C.vpx_image_t)(i),
		(C.vpx_img_fmt_t)(fmt),
		(C.uint)(dW),
		(C.uint)(dH),
		(C.uint)(align),
	))
}

func (i *Image) Read(data []byte) {
	C.vpx_img_read(
		(*C.vpx_image_t)(i),
		unsafe.Pointer(&data[0]),
	)
}

func (i *Image) Free() {
	C.vpx_img_free(
		(*C.vpx_image_t)(i),
	)
}

func RgbToYuv(rgba []byte, w uint32, h uint32) []byte {
	size := int(float32(w*h) * 1.5)
	yuv := make([]byte, size, size)
	pos := 0

	// Y plane
	for y := uint32(0); y < h; y++ {
		for x := uint32(0); x < w; x++ {
			i := y*w + x
			yuv[pos] = (byte)(((66*(int)(rgba[3*i]) + 129*(int)(rgba[3*i+1]) + 25*(int)(rgba[3*i+2])) >> 8) + 16)
			pos++
		}
	}

	// U plane
	for y := uint32(0); y < h; y += 2 {
		for x := uint32(0); x < w; x += 2 {
			i := y*w + x
			yuv[pos] = (byte)(((-38*(int)(rgba[3*i]) + -74*(int)(rgba[3*i+1]) + 112*(int)(rgba[3*i+2])) >> 8) + 128)
			pos++
		}
	}

	// V plane
	for y := uint32(0); y < h; y += 2 {
		for x := uint32(0); x < w; x += 2 {
			i := y*w + x
			yuv[pos] = (byte)(((112*(int)(rgba[3*i]) + -94*(int)(rgba[3*i+1]) + -18*(int)(rgba[3*i+2])) >> 8) + 128)
			pos++
		}
	}

	return yuv
}
