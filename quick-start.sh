#!/bin/bash
# Quick reference for running Venture with the level editor

cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║              VENTURE LEVEL EDITOR - QUICK START                  ║
╚══════════════════════════════════════════════════════════════════╝

📋 REQUIREMENTS
   ✓ CGAL installed (brew install cgal)
   ✓ C++ wrapper built (cd bsp/cgal && make)

🚀 RUN THE EDITOR

   Simplest method:
   $ ./run.sh level levels/test.yaml

   Alternative (manual):
   $ export DYLD_LIBRARY_PATH="${DYLD_LIBRARY_PATH}:${PWD}/bsp/cgal"
   $ go run . level levels/test.yaml

🛠️  COLLISION TEST TOOL

   1. Click "Collision Test" button (3rd tool)
   2. Click anywhere to test collision
   3. Continue clicking to trace lines
   
   Visual feedback:
   • Green circle = empty space
   • Red circle = solid
   • Pink line = trace path
   • Pink circle on line = intersection

📚 MORE INFO

   - RUNNING.md - Detailed running instructions
   - COLLISION_TEST_IMPLEMENTATION.md - Full implementation details
   - level/COLLISION_TEST_TOOL.md - Feature documentation
   - bsp/README.md - BSP package overview

🧪 TESTING

   $ cd bsp && ./run_tests.sh
   $ DYLD_LIBRARY_PATH="${PWD}/bsp/cgal" go test ./...

❓ TROUBLESHOOTING

   Error: "Library not loaded: libpartition.dylib"
   → Use ./run.sh or set DYLD_LIBRARY_PATH

   Error: "libpartition.dylib: no such file"
   → Build it: cd bsp/cgal && make

   Error: "CGAL not found"
   → Install it: brew install cgal

EOF

