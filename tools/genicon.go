//go:build ignore

// 图标生成器：go run tools/genicon.go  ->  build/icon.ico, icon.ico, icon_preview.png
// 用带符号距离场(SDF)做抗锯齿，绘制“端口/终止”主题图标：
// 深色圆角方块 + 红色电源符号(IEC power) + 绿色 >_ 终端角标。
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const SS = 1024

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

type rgba struct{ r, g, b, a float64 }

func over(dst, src rgba) rgba {
	sa, da := src.a, dst.a
	a := sa + da*(1-sa)
	if a == 0 {
		return rgba{}
	}
	r := (src.r*sa + dst.r*da*(1-sa)) / a
	g := (src.g*sa + dst.g*da*(1-sa)) / a
	b := (src.b*sa + dst.b*da*(1-sa)) / a
	return rgba{r, g, b, a}
}

func roundedBoxDist(px, py, cx, cy, hx, hy, r float64) float64 {
	qx := math.Abs(px-cx) - (hx - r)
	qy := math.Abs(py-cy) - (hy - r)
	ax := math.Max(qx, 0)
	ay := math.Max(qy, 0)
	return math.Hypot(ax, ay) + math.Min(math.Max(qx, qy), 0) - r
}

func segDist(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	t := clamp(((px-ax)*dx+(py-ay)*dy)/(dx*dx+dy*dy+1e-9), 0, 1)
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

func render() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, SS, SS))
	f := 3.0
	cx, cy := SS/2.0, SS/2.0
	tileH := SS * 0.46
	tileR := SS * 0.14
	for y := 0; y < SS; y++ {
		for x := 0; x < SS; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			col := rgba{0, 0, 0, 0}

			dTile := roundedBoxDist(px, py, cx, cy, tileH, tileH, tileR)
			aTile := clamp(0.5-dTile/f, 0, 1)
			if aTile > 0 {
				gt := clamp((py-(cy-tileH))/(2*tileH), 0, 1)
				bg := rgba{lerp(0x24, 0x0d, gt) / 255, lerp(0x2e, 0x12, gt) / 255, lerp(0x45, 0x1e, gt) / 255, aTile}
				edge := clamp((math.Abs(dTile)+SS*0.004)*0, 0, 1) // unused
				_ = edge
				ring := clamp((f - math.Abs(dTile+f*0.0) - (dTile*f)) , 0, 0) // noop
				_ = ring
				// 细亮边
				border := clamp(0.5+(-math.Abs(dTile))/f, 0, 1) * (1 - aTile*0) * (1 - clamp((dTile+f)/f, 0, 1))
				bg = over(bg, rgba{0.36, 0.44, 0.58, border * 0.95})
				col = over(col, bg)
			}

			// 红色电源符号环
			R := SS * 0.205
			th := SS * 0.058
			dc := math.Hypot(px-cx, py-cy)
			dRing := math.Abs(dc - R)
			ringCov := clamp((th/2-dRing)/f+0.5, 0, 1)
			topness := -(py - cy) / (dc + 1e-9)
			if topness > math.Cos(0.60) { // 顶部缺口
				ringCov = 0
			}
			col = over(col, rgba{0.96, 0.30, 0.32, ringCov})

			// 电源竖条
			bw := SS * 0.055
			barTop := cy - R*1.55
			barBot := cy - R*0.05
			dBar := math.Max(math.Abs(px-cx)-bw/2, math.Max(barTop-py, py-barBot))
			aBar := clamp(0.5-dBar/f, 0, 1)
			col = over(col, rgba{0.99, 0.99, 1.0, aBar})

			// 左下绿色 >_ 角标
			gx, gy := SS*0.30, SS*0.685
			sz := SS * 0.08
			pen := SS * 0.032
			d1 := segDist(px, py, gx-sz*0.45, gy-sz*0.8, gx+sz*0.55, gy)
			d2 := segDist(px, py, gx+sz*0.55, gy, gx-sz*0.45, gy+sz*0.8)
			aG := clamp(((pen/2-math.Min(d1, d2))/f)+0.5, 0, 1)
			col = over(col, rgba{0.42, 0.90, 0.52, aG})
			du := segDist(px, py, gx-sz*0.55, gy+sz*1.25, gx+sz*0.8, gy+sz*1.25)
			aU := clamp(((pen/2-du)/f)+0.5, 0, 1)
			col = over(col, rgba{0.42, 0.90, 0.52, aU})

			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(clamp(col.r*255, 0, 255) + 0.5),
				G: uint8(clamp(col.g*255, 0, 255) + 0.5),
				B: uint8(clamp(col.b*255, 0, 255) + 0.5),
				A: uint8(clamp(col.a*255, 0, 255) + 0.5),
			})
		}
	}
	return img
}

func downscale(src *image.NRGBA, n int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, n, n))
	factor := float64(src.Bounds().Dx()) / float64(n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var r, g, b, a float64
			y0 := int(float64(y) * factor)
			y1 := int(math.Max(float64(y0+1), float64(y+1)*factor))
			x0 := int(float64(x) * factor)
			x1 := int(math.Max(float64(x0+1), float64(x+1)*factor))
			for yy := y0; yy < y1 && yy < src.Bounds().Dy(); yy++ {
				for xx := x0; xx < x1 && xx < src.Bounds().Dx(); xx++ {
					c := src.NRGBAAt(xx, yy)
					al := float64(c.A)
					r += float64(c.R) * al
					g += float64(c.G) * al
					b += float64(c.B) * al
					a += al
				}
			}
			np := float64((y1 - y0) * (x1 - x0))
			nA := a / np
			var R, G, B float64
			if nA > 0 && a > 0 {
				R = r / a
				G = g / a
				B = b / a
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(clamp(R, 0, 255) + 0.5),
				G: uint8(clamp(G, 0, 255) + 0.5),
				B: uint8(clamp(B, 0, 255) + 0.5),
				A: uint8(clamp(nA, 0, 255) + 0.5),
			})
		}
	}
	return dst
}

func pngBytes(img image.Image) []byte {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func main() {
	os.MkdirAll("build", 0755)
	base := render()
	os.WriteFile("build/icon_preview.png", pngBytes(downscale(base, 256)), 0644)

	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	var imgs [][]byte
	for _, s := range sizes {
		imgs = append(imgs, pngBytes(downscale(base, s)))
	}
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint16(0))
	binary.Write(&out, binary.LittleEndian, uint16(1))
	binary.Write(&out, binary.LittleEndian, uint16(len(imgs)))
	offset := uint32(6 + 16*len(imgs))
	for i, d := range sizes {
		w, h := byte(d), byte(d)
		if d >= 256 {
			w, h = 0, 0
		}
		out.WriteByte(w)
		out.WriteByte(h)
		out.WriteByte(0)
		out.WriteByte(0)
		binary.Write(&out, binary.LittleEndian, uint16(1))
		binary.Write(&out, binary.LittleEndian, uint16(32))
		binary.Write(&out, binary.LittleEndian, uint32(len(imgs[i])))
		binary.Write(&out, binary.LittleEndian, offset)
		offset += uint32(len(imgs[i]))
	}
	for _, d := range imgs {
		out.Write(d)
	}
	os.WriteFile("build/icon.ico", out.Bytes(), 0644)
	os.WriteFile("icon.ico", out.Bytes(), 0644)
	os.WriteFile("icon_preview.png", pngBytes(downscale(base, 256)), 0644)
	os.MkdirAll("icons", 0755)
	for i, s := range sizes {
		os.WriteFile(fmt.Sprintf("icons/icon%d.png", s), imgs[i], 0644)
	}
	println("wrote icon.ico, build/icon.ico, icon_preview.png, icons/icon{16..256}.png")
}
