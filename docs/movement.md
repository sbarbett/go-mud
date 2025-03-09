---
title: Movement
keywords: movement, travel, directions, north, south, east, west, up, down
---
# Movement System

Moving around in Go-MUD is done using cardinal direction commands.

## Basic Movement Commands

- `north` or `n` - Move north
- `south` or `s` - Move south
- `east` or `e` - Move east
- `west` or `w` - Move west
- `up` or `u` - Move up
- `down` or `d` - Move down

## Example

```
> look
Market Square
A bustling market square with vendors selling various goods.
Exits: north, east, south, west

> north
You move north.

> look
Town Hall
The grand town hall stands before you, its doors open to the public.
Exits: south
```

## Movement Restrictions

Your movement may be restricted by:
- Locked doors
- Terrain obstacles
- Being in combat
- Being dead

If you're in combat, you must successfully `flee` before you can move.
If you're dead, you must `respawn` before you can move.

## Special Movement

The `recall` command will instantly transport you back to the starting area, regardless of your current location. This can be useful if you get lost or stuck. 