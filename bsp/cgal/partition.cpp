#include "partition.h"
#include <CGAL/Exact_predicates_inexact_constructions_kernel.h>
#include <CGAL/Partition_traits_2.h>
#include <CGAL/partition_2.h>
#include <CGAL/Polygon_2.h>
#include <vector>
#include <list>
#include <cstring>
#include <cstdlib>

typedef CGAL::Exact_predicates_inexact_constructions_kernel K;
typedef CGAL::Partition_traits_2<K> Traits;
typedef Traits::Point_2 Point_2;
typedef Traits::Polygon_2 Polygon_2;
typedef std::list<Polygon_2> Polygon_list;

// Helper function to allocate error string
static char* alloc_error(const char* msg) {
    char* err = (char*)malloc(strlen(msg) + 1);
    if (err) {
        strcpy(err, msg);
    }
    return err;
}

extern "C" {

CPartitionResult partition_polygon_convex(const CPoint* points, int count) {
    CPartitionResult result = {NULL, 0, NULL};
    
    // Validate input
    if (points == NULL || count < 3) {
        result.error = alloc_error("Invalid input: need at least 3 points");
        return result;
    }
    
    try {
        // Convert C points to CGAL polygon
        Polygon_2 polygon;
        for (int i = 0; i < count; i++) {
            polygon.push_back(Point_2(points[i].x, points[i].y));
        }
        
        // Check if polygon is valid (simple and non-degenerate)
        if (!polygon.is_simple()) {
            result.error = alloc_error("Polygon is not simple (self-intersecting)");
            return result;
        }
        
        // Check orientation - CGAL partition requires counter-clockwise
        if (polygon.is_clockwise_oriented()) {
            polygon.reverse_orientation();
        }
        
        // If already convex, return as-is
        if (polygon.is_convex()) {
            result.polygons = (CPolygon*)malloc(sizeof(CPolygon));
            if (!result.polygons) {
                result.error = alloc_error("Memory allocation failed");
                return result;
            }
            
            result.count = 1;
            result.polygons[0].count = count;
            result.polygons[0].points = (CPoint*)malloc(count * sizeof(CPoint));
            if (!result.polygons[0].points) {
                free(result.polygons);
                result.polygons = NULL;
                result.count = 0;
                result.error = alloc_error("Memory allocation failed");
                return result;
            }
            
            for (int i = 0; i < count; i++) {
                result.polygons[0].points[i] = points[i];
            }
            
            return result;
        }
        
        // Partition into convex sub-polygons
        Polygon_list partition_polys;
        CGAL::approx_convex_partition_2(polygon.vertices_begin(), 
                                       polygon.vertices_end(),
                                       std::back_inserter(partition_polys));
        
        // If partition failed or is empty, return error
        if (partition_polys.empty()) {
            result.error = alloc_error("Partition failed: no polygons generated");
            return result;
        }
        
        // Convert result back to C structures
        result.count = partition_polys.size();
        result.polygons = (CPolygon*)malloc(result.count * sizeof(CPolygon));
        if (!result.polygons) {
            result.error = alloc_error("Memory allocation failed");
            result.count = 0;
            return result;
        }
        
        int poly_idx = 0;
        for (const auto& part_poly : partition_polys) {
            int n = part_poly.size();
            result.polygons[poly_idx].count = n;
            result.polygons[poly_idx].points = (CPoint*)malloc(n * sizeof(CPoint));
            
            if (!result.polygons[poly_idx].points) {
                // Clean up previously allocated polygons
                for (int j = 0; j < poly_idx; j++) {
                    free(result.polygons[j].points);
                }
                free(result.polygons);
                result.polygons = NULL;
                result.count = 0;
                result.error = alloc_error("Memory allocation failed");
                return result;
            }
            
            int pt_idx = 0;
            for (auto vit = part_poly.vertices_begin(); vit != part_poly.vertices_end(); ++vit) {
                result.polygons[poly_idx].points[pt_idx].x = CGAL::to_double(vit->x());
                result.polygons[poly_idx].points[pt_idx].y = CGAL::to_double(vit->y());
                pt_idx++;
            }
            
            poly_idx++;
        }
        
        return result;
        
    } catch (const std::exception& e) {
        result.error = alloc_error(e.what());
        return result;
    } catch (...) {
        result.error = alloc_error("Unknown error during partition");
        return result;
    }
}

void free_partition_result(CPartitionResult* result) {
    if (result == NULL) {
        return;
    }
    
    if (result->polygons != NULL) {
        for (int i = 0; i < result->count; i++) {
            if (result->polygons[i].points != NULL) {
                free(result->polygons[i].points);
            }
        }
        free(result->polygons);
        result->polygons = NULL;
    }
    
    if (result->error != NULL) {
        free(result->error);
        result->error = NULL;
    }
    
    result->count = 0;
}

} // extern "C"

