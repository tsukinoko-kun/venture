# Makefile for building CGAL partition library
# This library is used via CGO from Go code

CXX = clang++
CXXFLAGS = -std=c++17 -O2 -Wall -fPIC
LDFLAGS = -shared

# CGAL and GMP flags
CGAL_CXXFLAGS = $(shell pkg-config --cflags cgal 2>/dev/null || echo "-I/opt/homebrew/include")
CGAL_LDFLAGS = $(shell pkg-config --libs cgal 2>/dev/null || echo "-L/opt/homebrew/lib -lgmp")

TARGET = libpartition.dylib
TARGET_LINUX = libpartition.so
TARGET_STATIC = libpartition.a

SOURCES = partition.cpp
OBJECTS = $(SOURCES:.cpp=.o)

# Default target for macOS
ifeq ($(shell uname -s),Darwin)
    TARGET_SHARED = $(TARGET)
else
    TARGET_SHARED = $(TARGET_LINUX)
endif

.PHONY: all clean static shared

all: $(TARGET_SHARED)

static: $(TARGET_STATIC)

shared: $(TARGET_SHARED)

$(TARGET): $(OBJECTS)
	$(CXX) $(LDFLAGS) -o $@ $^ $(CGAL_LDFLAGS)

$(TARGET_LINUX): $(OBJECTS)
	$(CXX) $(LDFLAGS) -o $@ $^ $(CGAL_LDFLAGS)

$(TARGET_STATIC): $(OBJECTS)
	ar rcs $@ $^

%.o: %.cpp partition.h
	$(CXX) $(CXXFLAGS) $(CGAL_CXXFLAGS) -c $< -o $@

clean:
	rm -f $(OBJECTS) $(TARGET) $(TARGET_LINUX) $(TARGET_STATIC)

# Test compilation
test: partition_test.cpp $(TARGET_SHARED)
	$(CXX) $(CXXFLAGS) $(CGAL_CXXFLAGS) -o partition_test partition_test.cpp -L. -lpartition $(CGAL_LDFLAGS)
	@echo "Test executable created: ./partition_test"

