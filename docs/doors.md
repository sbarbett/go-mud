---
title: Doors
keywords: doors, door, open, closed, locked, gate, gates, close
---

# Doors

In the world of Go-MUD, some passages between rooms are blocked by doors. These doors can be opened to allow passage.

## Identifying Doors

Doors are shown in the room's exit list with parentheses around the direction:

```
Available exits: [north, (west)]
```

In this example, the west exit has a closed door.

## Opening and Closing Doors

To open a door, you can use the `open` command followed by either:

1. The direction of the door: `open west`
2. The keyword for the door: `open gate`

Similarly, to close a door, use the `close` command:

1. The direction of the door: `close west`
2. The keyword for the door: `close gate`

## Door States

Doors can be in one of three states:

- **Open**: You can freely pass through the door.
- **Closed**: The door blocks passage but can be opened.
- **Locked**: The door is closed and cannot be opened without a key (not yet implemented).

## Door Synchronization

Doors in Go-MUD are synchronized between connected rooms. This means:

- If you open or close a door in one room, it will also be open or closed when viewed from the connected room.
- When a door automatically closes, it closes on both sides.
- Players in both rooms will be notified when a door is opened or closed.

For example, if you open a gate to the west in Room A, the gate to the east in Room B will also be open.

## Notes

- Doors will automatically close after some time (approximately 15 minutes).
- If a door is locked, you will need to find a key to unlock it (future feature).
- You cannot pass through a closed door without opening it first.

## Related Commands

- `open <direction/keyword>` - Opens a door
- `close <direction/keyword>` - Closes a door 