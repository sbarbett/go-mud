---
title: Goto
keywords: goto, teleport, room, id, admin, debug
---
# Goto Command

The `goto` command allows instant teleportation to any room in the game by specifying its room ID.

## Usage

```
goto <room_id>
```

Where `<room_id>` is the numeric ID of the room you want to teleport to.

## Examples

```
> goto 3001
You teleport to Room 3001 (Town Square).

> goto 9999
Room 9999 does not exist.
```

## Notes

- This command bypasses normal movement restrictions and allows instant travel to any valid room.
- You will not pass through any rooms between your current location and the destination.
- Doors, locks, and other movement restrictions are ignored.
- This is primarily an administrative/debugging command.
- If the specified room does not exist, you will receive an error message. 