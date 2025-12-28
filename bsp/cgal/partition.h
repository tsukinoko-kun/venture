#ifndef BSP_PARTITION_H
#define BSP_PARTITION_H

#ifdef __cplusplus
extern "C" {
#endif

// C-compatible point structure
typedef struct {
    double x;
    double y;
} CPoint;

// C-compatible polygon structure
typedef struct {
    CPoint* points;
    int count;
} CPolygon;

// Result structure containing array of partitioned polygons
typedef struct {
    CPolygon* polygons;
    int count;
    char* error; // NULL if success, error message otherwise
} CPartitionResult;

// Partition a polygon into convex sub-polygons
// Input: points array and count
// Output: CPartitionResult with convex polygons
// Caller must free the result using free_partition_result
CPartitionResult partition_polygon_convex(const CPoint* points, int count);

// Free memory allocated by partition_polygon_convex
void free_partition_result(CPartitionResult* result);

#ifdef __cplusplus
}
#endif

#endif // BSP_PARTITION_H

