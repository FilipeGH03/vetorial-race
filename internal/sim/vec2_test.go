package sim

import (
	"math"
	"testing"
)

func perto(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestVec2(t *testing.T) {
	casos := []struct {
		nome string
		got  Vec2
		want Vec2
	}{
		{"Add", Vec2{1, 2}.Add(Vec2{3, -4}), Vec2{4, -2}},
		{"Sub", Vec2{1, 2}.Sub(Vec2{3, -4}), Vec2{-2, 6}},
		{"Scale", Vec2{1, -2}.Scale(3), Vec2{3, -6}},
		{"Norm", Vec2{3, 4}.Norm(), Vec2{0.6, 0.8}},
		{"Norm nulo", Vec2{}.Norm(), Vec2{}},
		{"Rotate 90", Vec2{1, 0}.Rotate(math.Pi / 2), Vec2{0, 1}},
		{"Rotate 180", Vec2{1, 2}.Rotate(math.Pi), Vec2{-1, -2}},
	}
	for _, c := range casos {
		if !perto(c.got.X, c.want.X) || !perto(c.got.Y, c.want.Y) {
			t.Errorf("%s: got %v, want %v", c.nome, c.got, c.want)
		}
	}
}

func TestVec2Escalares(t *testing.T) {
	casos := []struct {
		nome string
		got  float64
		want float64
	}{
		{"Dot", Vec2{1, 2}.Dot(Vec2{3, 4}), 11},
		{"Dot ortogonal", Vec2{1, 0}.Dot(Vec2{0, 5}), 0},
		{"Cross x,y", Vec2{1, 0}.Cross(Vec2{0, 1}), 1},
		{"Cross y,x", Vec2{0, 1}.Cross(Vec2{1, 0}), -1},
		{"Len", Vec2{3, 4}.Len(), 5},
		{"Angle (1,0)", Vec2{1, 0}.Angle(), 0},
		{"Angle (0,1)", Vec2{0, 1}.Angle(), math.Pi / 2},
		{"Angle (-1,0)", Vec2{-1, 0}.Angle(), math.Pi},
		{"Angle (0,-1)", Vec2{0, -1}.Angle(), -math.Pi / 2},
		{"Angle (1,1)", Vec2{1, 1}.Angle(), math.Pi / 4},
	}
	for _, c := range casos {
		if !perto(c.got, c.want) {
			t.Errorf("%s: got %v, want %v", c.nome, c.got, c.want)
		}
	}
}
