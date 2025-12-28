#include <stdio.h>
#include "partition.h"

int main() {
    // Test with a simple L-shaped polygon (concave)
    CPoint points[] = {
        {0, 0},
        {4, 0},
        {4, 2},
        {2, 2},
        {2, 4},
        {0, 4}
    };
    int count = 6;
    
    printf("Testing partition with L-shaped polygon (%d points)...\n", count);
    
    CPartitionResult result = partition_polygon_convex(points, count);
    
    if (result.error != NULL) {
        printf("ERROR: %s\n", result.error);
        free_partition_result(&result);
        return 1;
    }
    
    printf("Success! Partitioned into %d convex polygon(s)\n", result.count);
    
    for (int i = 0; i < result.count; i++) {
        printf("  Polygon %d: %d vertices\n", i + 1, result.polygons[i].count);
        for (int j = 0; j < result.polygons[i].count; j++) {
            printf("    (%f, %f)\n", 
                   result.polygons[i].points[j].x,
                   result.polygons[i].points[j].y);
        }
    }
    
    free_partition_result(&result);
    
    // Test with a convex square (should return as-is)
    printf("\nTesting with convex square...\n");
    CPoint square[] = {
        {0, 0},
        {1, 0},
        {1, 1},
        {0, 1}
    };
    
    result = partition_polygon_convex(square, 4);
    
    if (result.error != NULL) {
        printf("ERROR: %s\n", result.error);
        free_partition_result(&result);
        return 1;
    }
    
    printf("Success! Square partitioned into %d polygon(s) (should be 1)\n", result.count);
    free_partition_result(&result);
    
    printf("\nAll tests passed!\n");
    return 0;
}

