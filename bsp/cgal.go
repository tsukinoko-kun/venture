package bsp

/*
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/cgal
#cgo darwin LDFLAGS: ${SRCDIR}/cgal/libpartition.a -L/opt/homebrew/lib -lgmp -lc++
#cgo linux LDFLAGS: ${SRCDIR}/cgal/libpartition.a -lgmp -lstdc++
#cgo windows LDFLAGS: ${SRCDIR}/cgal/libpartition.a -lgmp -lstdc++
#include "cgal/partition.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// PartitionPolygonConvex takes a polygon and partitions it into convex sub-polygons
// using CGAL's approx_convex_partition_2 algorithm.
// If the polygon is already convex, it returns it as-is.
// Returns a slice of convex polygons or an error.
func PartitionPolygonConvex(polygon Polygon) ([]Polygon, error) {
	if len(polygon.Vertices) < 3 {
		return nil, fmt.Errorf("polygon must have at least 3 vertices")
	}

	// Convert Go points to C points
	cPoints := make([]C.CPoint, len(polygon.Vertices))
	for i, v := range polygon.Vertices {
		cPoints[i].x = C.double(v.X)
		cPoints[i].y = C.double(v.Y)
	}

	// Call C function
	result := C.partition_polygon_convex(&cPoints[0], C.int(len(cPoints)))
	defer C.free_partition_result(&result)

	// Check for errors
	if result.error != nil {
		errMsg := C.GoString(result.error)
		return nil, fmt.Errorf("CGAL partition error: %s", errMsg)
	}

	// Convert result back to Go polygons
	if result.count == 0 {
		return nil, fmt.Errorf("partition returned no polygons")
	}

	// Access the C array of polygons
	cPolygons := (*[1 << 30]C.CPolygon)(unsafe.Pointer(result.polygons))[:result.count:result.count]

	goPolygons := make([]Polygon, result.count)
	for i := 0; i < int(result.count); i++ {
		cPoly := cPolygons[i]

		// Access the C array of points for this polygon
		cPolyPoints := (*[1 << 30]C.CPoint)(unsafe.Pointer(cPoly.points))[:cPoly.count:cPoly.count]

		vertices := make([]Point, cPoly.count)
		for j := 0; j < int(cPoly.count); j++ {
			vertices[j] = Point{
				X: float32(cPolyPoints[j].x),
				Y: float32(cPolyPoints[j].y),
			}
		}

		goPolygons[i] = Polygon{
			Vertices: vertices,
			IsSolid:  polygon.IsSolid, // Preserve solid flag from original
		}
	}

	return goPolygons, nil
}
