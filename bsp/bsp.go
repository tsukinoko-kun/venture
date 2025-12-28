package bsp

import (
	"math"

	pb "github.com/bloodmagesoftware/venture/proto/level"
)

// Point represents a 2D point
type Point struct {
	X, Y float32
}

// Polygon represents a collision polygon
type Polygon struct {
	Vertices []Point
	IsSolid  bool // true for solid walls, false for empty space
}

// Vector2 represents a 2D vector
type Vector2 struct {
	X, Y float32
}

// Normalize returns a normalized copy of the vector
func (v Vector2) Normalize() Vector2 {
	length := float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
	if length == 0 {
		return Vector2{0, 0}
	}
	return Vector2{X: v.X / length, Y: v.Y / length}
}

// Dot returns the dot product of two vectors
func (v Vector2) Dot(other Vector2) float32 {
	return v.X*other.X + v.Y*other.Y
}

// Line represents a 2D line using the plane equation: Normal · Point = Distance
type Line struct {
	Normal   Vector2
	Distance float32
}

// PointSide determines which side of the line a point is on
// Returns: > 0 for front, < 0 for back, 0 for on the line
func (l Line) PointSide(p Point) float32 {
	return l.Normal.X*p.X + l.Normal.Y*p.Y - l.Distance
}

// ClassifyPoint returns 1 for front, -1 for back, 0 for on the line
func (l Line) ClassifyPoint(p Point) int {
	side := l.PointSide(p)
	epsilon := float32(0.0001)
	if side > epsilon {
		return 1 // Front
	} else if side < -epsilon {
		return -1 // Back
	}
	return 0 // On line
}

// BSPBuilder holds the state for building a BSP tree
type BSPBuilder struct {
	Polygons []Polygon
}

// NewBSPBuilder creates a new BSP builder with the given polygons
func NewBSPBuilder(polygons []Polygon) *BSPBuilder {
	return &BSPBuilder{
		Polygons: polygons,
	}
}

// Build constructs the BSP tree and returns the root node
// This is where you'll implement the actual BSP construction algorithm
func (b *BSPBuilder) Build() *pb.BSPNode {
	// TODO: Implement BSP tree construction
	// For now, return a simple leaf node
	return &pb.BSPNode{
		Type: &pb.BSPNode_Leaf{
			Leaf: &pb.Leaf{
				SectorId:       0,
				PolygonIndices: []int32{},
				IsSolid:        false,
			},
		},
	}
}

// PointInBSP tests if a point is inside solid geometry using the BSP tree
func PointInBSP(node *pb.BSPNode, point Point) bool {
	if node == nil {
		return false
	}

	switch n := node.Type.(type) {
	case *pb.BSPNode_Leaf:
		// Leaf node: return the solid state
		return n.Leaf.IsSolid

	case *pb.BSPNode_Split:
		// Split node: determine which side and recurse
		split := n.Split
		line := Line{
			Normal:   Vector2{X: split.NormalX, Y: split.NormalY},
			Distance: split.Distance,
		}

		side := line.PointSide(point)
		if side >= 0 {
			// Point is on the front side
			return PointInBSP(split.Front, point)
		} else {
			// Point is on the back side
			return PointInBSP(split.Back, point)
		}

	default:
		return false
	}
}

// Helper functions for creating protobuf nodes

// NewLeafNode creates a new leaf node
func NewLeafNode(sectorID int32, polygonIndices []int32, isSolid bool) *pb.BSPNode {
	return &pb.BSPNode{
		Type: &pb.BSPNode_Leaf{
			Leaf: &pb.Leaf{
				SectorId:       sectorID,
				PolygonIndices: polygonIndices,
				IsSolid:        isSolid,
			},
		},
	}
}

// NewSplitNode creates a new split node
func NewSplitNode(normalX, normalY, distance float32, front, back *pb.BSPNode) *pb.BSPNode {
	return &pb.BSPNode{
		Type: &pb.BSPNode_Split{
			Split: &pb.Split{
				NormalX:  normalX,
				NormalY:  normalY,
				Distance: distance,
				Front:    front,
				Back:     back,
			},
		},
	}
}

// NewLevelData creates a new level data protobuf with the given root node
func NewLevelData(root *pb.BSPNode) *pb.LevelData {
	return &pb.LevelData{
		Root: root,
	}
}
