// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General
// Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program.  If not, see <http://www.gnu.org/licenses/>.

package screengen

import (
	"image"
	"unsafe"
)

func rotate90(m *image.RGBA) *image.RGBA {
	w := m.Rect.Max.X - m.Rect.Min.X
	h := m.Rect.Max.Y - m.Rect.Min.Y
	d := image.NewRGBA(image.Rect(0, 0, h, w))
	dst := d.Pix
	src := m.Pix
	s1 := m.Stride
	s2 := h * 4
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := x*s2 + (h-y-1)*4
			j := y*s1 + x*4
			*(*uint32)(unsafe.Pointer(&dst[i])) = *(*uint32)(unsafe.Pointer(&src[j]))
		}
	}
	return d
}

func rotate270(m *image.RGBA) *image.RGBA {
	w := m.Rect.Max.X - m.Rect.Min.X
	h := m.Rect.Max.Y - m.Rect.Min.Y
	d := image.NewRGBA(image.Rect(0, 0, h, w))
	dst := d.Pix
	src := m.Pix
	s1 := m.Stride
	s2 := h * 4
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (w-x-1)*s2 + y*4
			j := y*s1 + x*4
			*(*uint32)(unsafe.Pointer(&dst[i])) = *(*uint32)(unsafe.Pointer(&src[j]))
		}
	}
	return d
}

func rotate180(m *image.RGBA) *image.RGBA {
	buf := make([]uint8, m.Stride)
	i := 0
	j := len(m.Pix) - m.Stride
	for i < j {
		copy(buf, m.Pix[i:])
		copy(m.Pix[i:], m.Pix[j:j+m.Stride])
		copy(m.Pix[j:], buf)
		flip(m.Pix[i : i+m.Stride])
		flip(m.Pix[j : j+m.Stride])
		i += m.Stride
		j -= m.Stride
	}
	if i == j {
		flip(m.Pix[i : i+m.Stride])
	}
	return m
}

func flip(pix []uint8) {
	i := 0
	j := len(pix) - 4
	for i < j {
		p1 := (*uint32)(unsafe.Pointer(&pix[i]))
		p2 := (*uint32)(unsafe.Pointer(&pix[j]))
		*p1, *p2 = *p2, *p1
		i += 4
		j -= 4
	}
}
