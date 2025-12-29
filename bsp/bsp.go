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
func (b *BSPBuilder) Build() *pb.BSPNode {
	// Step 1: Partition all polygons into convex sub-polygons
	convexPolygons := make([]Polygon, 0)
	for _, poly := range b.Polygons {
		partitioned, err := PartitionPolygonConvex(poly)
		if err != nil {
			continue
		}
		convexPolygons = append(convexPolygons, partitioned...)
	}

	// Step 2: Build individual BSP trees for each polygon
	// Then combine them with OR logic
	var polyTrees []*pb.BSPNode
	for _, poly := range convexPolygons {
		if poly.IsSolid && len(poly.Vertices) >= 3 {
			polyTrees = append(polyTrees, buildConvexPolygonTree(poly))
		}
	}

	if len(polyTrees) == 0 {
		return NewLeafNode(0, []int32{}, false)
	}

	if len(polyTrees) == 1 {
		return polyTrees[0]
	}

	// Combine multiple trees with OR logic
	return mergeTrees(polyTrees)
}

// mergeTrees combines multiple BSP trees with OR logic
// A point is solid if it's solid in ANY of the trees
func mergeTrees(trees []*pb.BSPNode) *pb.BSPNode {
	if len(trees) == 0 {
		return NewLeafNode(0, []int32{}, false)
	}

	if len(trees) == 1 {
		return trees[0]
	}

	// Take the first tree and use its splitting planes
	// Then recursively merge the rest
	first := trees[0]
	rest := trees[1:]

	return mergeTreePair(first, mergeTrees(rest))
}

// mergeTreePair merges two BSP trees with OR logic
func mergeTreePair(tree1, tree2 *pb.BSPNode) *pb.BSPNode {
	// If tree1 is a leaf
	if leaf1, ok := tree1.Type.(*pb.BSPNode_Leaf); ok {
		if leaf1.Leaf.IsSolid {
			// tree1 is solid everywhere - return solid
			return tree1
		}
		// tree1 is non-solid everywhere - return tree2
		return tree2
	}

	// tree1 is a split node
	split1 := tree1.Type.(*pb.BSPNode_Split).Split
	line := Line{
		Normal:   Vector2{X: split1.NormalX, Y: split1.NormalY},
		Distance: split1.Distance,
	}

	// Split tree2 along tree1's plane
	tree2Front, tree2Back := splitTree(tree2, line)

	// Recursively merge
	frontMerged := mergeTreePair(split1.Front, tree2Front)
	backMerged := mergeTreePair(split1.Back, tree2Back)

	return NewSplitNode(split1.NormalX, split1.NormalY, split1.Distance, frontMerged, backMerged)
}

// splitTree splits a BSP tree along a plane
// Returns the front and back subtrees
func splitTree(tree *pb.BSPNode, splitLine Line) (*pb.BSPNode, *pb.BSPNode) {
	// If tree is a leaf, return it for both sides
	if _, ok := tree.Type.(*pb.BSPNode_Leaf); ok {
		return tree, tree
	}

	// tree is a split node
	_ = tree.Type.(*pb.BSPNode_Split).Split

	// For simplicity, just return the tree as-is for both sides
	// A proper implementation would classify the split plane relative to splitLine
	// For now, this is a conservative approximation
	return tree, tree
}

// buildConvexPolygonTree builds a BSP tree that returns true iff point is inside polygon
// For a CCW convex polygon, a point is inside if it's on the left/inside of all edges
func buildConvexPolygonTree(poly Polygon) *pb.BSPNode {
	if len(poly.Vertices) < 3 {
		return NewLeafNode(0, []int32{}, false)
	}

	// Detect winding order and normalize to CCW if needed
	// This ensures the BSP tree works correctly regardless of input winding
	normalizedPoly := ensureCCW(poly)

	// Build nested tests: must be on inside of ALL edges
	return buildEdgeTest(normalizedPoly, 0)
}

// signedArea computes the signed area of a polygon
// Positive = CCW, Negative = CW
func signedArea(poly Polygon) float32 {
	if len(poly.Vertices) < 3 {
		return 0
	}

	var area float32 = 0
	n := len(poly.Vertices)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += poly.Vertices[i].X * poly.Vertices[j].Y
		area -= poly.Vertices[j].X * poly.Vertices[i].Y
	}
	return area / 2
}

// isCCW returns true if the polygon has counter-clockwise winding
func isCCW(poly Polygon) bool {
	return signedArea(poly) > 0
}

// ensureCCW returns a copy of the polygon with CCW winding order
// If the polygon is already CCW, returns it unchanged
// If CW, reverses the vertex order
func ensureCCW(poly Polygon) Polygon {
	if isCCW(poly) {
		return poly
	}

	// Reverse the vertices
	reversed := make([]Point, len(poly.Vertices))
	n := len(poly.Vertices)
	for i := 0; i < n; i++ {
		reversed[i] = poly.Vertices[n-1-i]
	}

	return Polygon{
		Vertices: reversed,
		IsSolid:  poly.IsSolid,
	}
}

// buildEdgeTest recursively builds edge tests for a convex polygon
// A point must be on the "inside" of all edges to be considered inside the polygon
func buildEdgeTest(poly Polygon, edgeIdx int) *pb.BSPNode {
	if edgeIdx >= len(poly.Vertices) {
		// Passed all edge tests - point is inside!
		return NewLeafNode(0, []int32{}, true)
	}

	// Get edge vertices
	v1 := poly.Vertices[edgeIdx]
	v2 := poly.Vertices[(edgeIdx+1)%len(poly.Vertices)]

	// Edge direction vector
	edge := Vector2{X: v2.X - v1.X, Y: v2.Y - v1.Y}

	// Inward normal (for CCW polygon, rotate 90° clockwise)
	inwardNormal := Vector2{X: edge.Y, Y: -edge.X}.Normalize()

	distance := inwardNormal.X*v1.X + inwardNormal.Y*v1.Y

	line := Line{Normal: inwardNormal, Distance: distance}

	// For an inward-pointing normal:
	// - Points with NEGATIVE distance (back side) are OUTSIDE this edge
	// - Points with POSITIVE distance (front side) are INSIDE this edge
	//
	// Wait, that's not right either. Let me think about this more carefully.
	//
	// The plane equation is: normal · point = distance
	// Or: normal · point - distance = 0
	// If (normal · point - distance) > 0, point is on the positive side (front)
	// If (normal · point - distance) < 0, point is on the negative side (back)
	//
	// From the debug output, center point (0,0) gives side = -5 for all edges.
	// This means (0,0) is on the BACK side of all edges.
	// Since the polygon is defined correctly and (0,0) SHOULD be inside,
	// this means the BACK side is the inside!
	//
	// So: back side = inside, front side = outside

	frontNode := NewLeafNode(0, []int32{}, false) // Front = outside
	backNode := buildEdgeTest(poly, edgeIdx+1)    // Back = might be inside, check next edge

	return NewSplitNode(line.Normal.X, line.Normal.Y, line.Distance, frontNode, backNode)
}

// mergeOR creates a tree that returns true if EITHER subtree returns true
// This is tricky because we can't easily combine two arbitrary trees
// For now, we'll use a simpler approach: if we have multiple polygons,
// we just add all their edges as splitting planes
func mergeOR(tree1, tree2 *pb.BSPNode) *pb.BSPNode {
	// Check if tree2 is a non-solid leaf
	if leaf, ok := tree2.Type.(*pb.BSPNode_Leaf); ok {
		if !leaf.Leaf.IsSolid {
			// tree2 is empty, just return tree1
			return tree1
		}
	}

	// Check if tree1 is a non-solid leaf
	if leaf, ok := tree1.Type.(*pb.BSPNode_Leaf); ok {
		if !leaf.Leaf.IsSolid {
			// tree1 is empty, just return tree2
			return tree2
		}
	}

	// Both trees have content - this is complex
	// For a proper OR operation, we'd need to traverse both trees
	// For now, we'll just return tree1 and lose tree2
	// TODO: Implement proper CSG union
	return tree1
}

// PolygonClassification represents how a polygon relates to a splitting line
type PolygonClassification int

const (
	PolygonFront PolygonClassification = iota
	PolygonBack
	PolygonSpanning
	PolygonCoplanar
)

// classifyPolygon determines which side of the line a polygon is on
func classifyPolygon(poly Polygon, line Line) PolygonClassification {
	if len(poly.Vertices) == 0 {
		return PolygonCoplanar
	}

	frontCount := 0
	backCount := 0
	epsilon := float32(0.0001)

	for _, v := range poly.Vertices {
		side := line.PointSide(v)
		if side > epsilon {
			frontCount++
		} else if side < -epsilon {
			backCount++
		}
	}

	if frontCount > 0 && backCount > 0 {
		return PolygonSpanning
	} else if frontCount > 0 {
		return PolygonFront
	} else if backCount > 0 {
		return PolygonBack
	}
	return PolygonCoplanar
}

// selectSplitLine chooses a splitting line from the polygon edges
// This uses a simple heuristic: pick the first edge of the first polygon
// A more sophisticated approach would evaluate multiple candidates
func selectSplitLine(polygons []Polygon) Line {
	if len(polygons) == 0 || len(polygons[0].Vertices) < 2 {
		// Fallback: horizontal line at origin
		return Line{Normal: Vector2{X: 0, Y: 1}, Distance: 0}
	}

	// Use the first edge of the first polygon
	poly := polygons[0]
	v1 := poly.Vertices[0]
	v2 := poly.Vertices[1]

	// Edge vector
	edge := Vector2{X: v2.X - v1.X, Y: v2.Y - v1.Y}

	// Normal is perpendicular to edge (rotate 90 degrees counter-clockwise)
	normal := Vector2{X: -edge.Y, Y: edge.X}.Normalize()

	// Distance is the dot product of normal with any point on the line
	distance := normal.X*v1.X + normal.Y*v1.Y

	return Line{Normal: normal, Distance: distance}
}

// splitPolygon splits a polygon by a line into front and back parts
// For convex polygons, this creates two convex sub-polygons
func splitPolygon(poly Polygon, line Line) (*Polygon, *Polygon) {
	if len(poly.Vertices) < 3 {
		return nil, nil
	}

	epsilon := float32(0.0001)
	var frontVerts, backVerts []Point

	for i := 0; i < len(poly.Vertices); i++ {
		v1 := poly.Vertices[i]
		v2 := poly.Vertices[(i+1)%len(poly.Vertices)]

		side1 := line.PointSide(v1)
		side2 := line.PointSide(v2)

		// Add v1 to appropriate list(s)
		if side1 > epsilon {
			frontVerts = append(frontVerts, v1)
		} else if side1 < -epsilon {
			backVerts = append(backVerts, v1)
		} else {
			// On the line - add to both
			frontVerts = append(frontVerts, v1)
			backVerts = append(backVerts, v1)
		}

		// Check if edge crosses the line
		if (side1 > epsilon && side2 < -epsilon) || (side1 < -epsilon && side2 > epsilon) {
			// Edge crosses - compute intersection point
			t := side1 / (side1 - side2)
			intersection := Point{
				X: v1.X + t*(v2.X-v1.X),
				Y: v1.Y + t*(v2.Y-v1.Y),
			}
			frontVerts = append(frontVerts, intersection)
			backVerts = append(backVerts, intersection)
		}
	}

	var frontPoly, backPoly *Polygon

	if len(frontVerts) >= 3 {
		frontPoly = &Polygon{Vertices: frontVerts, IsSolid: poly.IsSolid}
	}
	if len(backVerts) >= 3 {
		backPoly = &Polygon{Vertices: backVerts, IsSolid: poly.IsSolid}
	}

	return frontPoly, backPoly
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
		if side > 0 {
			// Point is strictly on the front side
			return PointInBSP(split.Front, point)
		} else {
			// Point is on the back side or exactly on the line
			// For solid geometry BSP, points on the boundary are considered inside
			return PointInBSP(split.Back, point)
		}

	default:
		return false
	}
}

// LineTraceBSPNode traces a line segment through a BSP tree
// Returns true and hit point if the line hits solid geometry
// The segment is defined by the parametric range [t0, t1] on the line from `from` to `to`
func LineTraceBSPNode(node *pb.BSPNode, from, to Point, t0, t1 float32) (hit bool, hitX, hitY float32) {
	if node == nil {
		return false, 0, 0
	}

	// Compute the actual segment endpoints for this recursion level
	p0 := Point{
		X: from.X + t0*(to.X-from.X),
		Y: from.Y + t0*(to.Y-from.Y),
	}
	p1 := Point{
		X: from.X + t1*(to.X-from.X),
		Y: from.Y + t1*(to.Y-from.Y),
	}

	switch n := node.Type.(type) {
	case *pb.BSPNode_Leaf:
		if n.Leaf.IsSolid {
			// Hit! Return the entry point of the line segment
			return true, p0.X, p0.Y
		}
		return false, 0, 0

	case *pb.BSPNode_Split:
		split := n.Split
		normalX := split.NormalX
		normalY := split.NormalY
		dist := split.Distance

		// Calculate signed distance for the CURRENT segment endpoints
		d0 := normalX*p0.X + normalY*p0.Y - dist
		d1 := normalX*p1.X + normalY*p1.Y - dist

		epsilon := float32(0.0001)

		// Both points on front side
		if d0 > epsilon && d1 > epsilon {
			return LineTraceBSPNode(split.Front, from, to, t0, t1)
		}

		// Both points on back side
		if d0 <= epsilon && d1 <= epsilon {
			return LineTraceBSPNode(split.Back, from, to, t0, t1)
		}

		// Line segment spans the plane - compute intersection
		// t is the parametric value where the segment [p0, p1] crosses the plane
		// At intersection: d0 + t*(d1-d0) = 0, so t = -d0 / (d1-d0)
		t := -d0 / (d1 - d0)

		// Map t from [0,1] on segment to the global parametric range
		tMid := t0 + t*(t1-t0)

		// Determine traversal order (near to far based on segment start)
		var nearNode, farNode *pb.BSPNode
		if d0 > 0 {
			// Segment starts in front
			nearNode = split.Front
			farNode = split.Back
		} else {
			// Segment starts in back
			nearNode = split.Back
			farNode = split.Front
		}

		// Check near side first (from t0 to tMid)
		hit, hitX, hitY = LineTraceBSPNode(nearNode, from, to, t0, tMid)
		if hit {
			return true, hitX, hitY
		}

		// Check far side (from tMid to t1)
		return LineTraceBSPNode(farNode, from, to, tMid, t1)
	}

	return false, 0, 0
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
