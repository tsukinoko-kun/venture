# Makefile for building CGAL partition library
# This library is used via CGO from Go code

# Detect OS and set appropriate compiler
UNAME_S := $(shell uname -s 2>/dev/null || echo Windows)

ifeq ($(UNAME_S),Darwin)
    CXX = clang++
    CGAL_CXXFLAGS = $(shell pkg-config --cflags cgal 2>/dev/null || echo "-I/opt/homebrew/include")
    CGAL_LDFLAGS = $(shell pkg-config --libs cgal 2>/dev/null || echo "-L/opt/homebrew/lib -lgmp")
    TARGET_SHARED = libpartition.dylib
else ifeq ($(UNAME_S),Linux)
    CXX = g++
    CGAL_CXXFLAGS = $(shell pkg-config --cflags cgal 2>/dev/null || echo "")
    CGAL_LDFLAGS = $(shell pkg-config --libs cgal 2>/dev/null || echo "-lgmp")
    TARGET_SHARED = libpartition.so
else
    # Windows (MinGW)
    CXX = g++
    CGAL_CXXFLAGS = 
    CGAL_LDFLAGS = -lgmp
    TARGET_SHARED = libpartition.dll
endif

CXXFLAGS = -std=c++17 -O2 -Wall -fPIC
LDFLAGS = -shared

TARGET_STATIC = libpartition.a

SOURCES = partition.cpp
OBJECTS = $(SOURCES:.cpp=.o)

.PHONY: all clean static shared

# Build static library by default for Go integration
all: $(TARGET_STATIC)

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

