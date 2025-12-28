package level

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type (
	Level struct {
		// Ground is the visual base of the level. It is for grass, rock, etc.
		// Used for visuals only, not for collision detection.
		Ground []Tile `yaml:"ground"`
		// Objects is a lost of visual objects that are layered on top of the ground.
		Objects []Object `yaml:"objects"`
		// Collisions is a list of collision shapes (polygons).
		// Those are not visual, only for collision detection of static objects like walls, buildings, trees, etc.
		Collisions []Polygon `yaml:"collisions"`
		// Spawns is a list of spawn points.
		// Every spawn point has a name (map key) to reference it from other levels.
		Spawns map[string]Spawn `yaml:"spawns"`
		// Portals is a list of possible teleportation trigger points.
		// When a player walks near them, they will be teleported to the specified level and spawn point.
		Portals []Portal `yaml:"portals"`
	}

	Spawn struct {
		Position Vec2 `yaml:"position"`
	}

	Portal struct {
		// Position is the center of the portal.
		// This is the position in the level where the player is currently standing in.
		Position Vec2 `yaml:"position"`
		// Level is the level to teleport to.
		Level string `yaml:"level"`
		// Spawn is the spawn point to teleport to that must be defined in the level to teleport to.
		Spawn string `yaml:"spawn"`
	}

	Polygon struct {
		// Outline is a list of points that define the outer shape of a polygon.
		Outline Outline `yaml:"outline"`
	}

	Outline []Vec2

	Object struct {
		// Position is the center of the object.
		Position Vec2 `yaml:"position"`
		// Rotation is the rotation of the object in degrees.
		Rotation float64 `yaml:"rotation"`
		// Size is the size of the object in world units.
		// By default, 256px is 1 world unit.
		Size    Vec2   `yaml:"size"`
		Texture string `yaml:"texture"`
	}

	Vec2 struct {
		X float32 `yaml:"x"`
		Y float32 `yaml:"y"`
	}

	Vec2i struct {
		X int32 `yaml:"x"`
		Y int32 `yaml:"y"`
	}

	Tile struct {
		Position Vec2i  `yaml:"position"`
		Texture  string `yaml:"texture"`
	}
)

func New() *Level {
	return &Level{
		Ground:     make([]Tile, 0),
		Collisions: make([]Polygon, 0),
		Spawns:     make(map[string]Spawn),
		Portals:    make([]Portal, 0),
	}
}

func (l *Level) Save(path string) error {
	_ = os.MkdirAll(filepath.Dir(path), 0755)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	defer encoder.Close()
	encoder.SetIndent(4)

	return encoder.Encode(l)
}

func (l *Level) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	return decoder.Decode(l)
}
