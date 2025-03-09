---
title: Combat System
keywords: fighting, attack, defense, kill, flee
---
# Combat System

Combat in Go-MUD is turn-based and happens automatically once initiated.

## Starting Combat

To begin combat with a mob (NPC), use:
```
attack <mob name>
```
or
```
kill <mob name>
```

For example: `attack goblin` or `kill orc warrior`

## Combat Mechanics

Once combat begins:
- Every second, you will automatically attempt an attack
- The success of your attack depends on your stats and the enemy's defense
- Damage is calculated based on your strength and weapon
- Combat continues until either you or your opponent reaches 0 HP

## Fleeing from Combat

If a battle is going poorly, you can attempt to flee:
```
flee
```

Fleeing is not guaranteed to succeed. The chance depends on your agility compared to your opponent's.

## Checking Combat Status

To check your current combat status:
```
status
```
or
```
combat
```

This will show your health, your opponent's health, and other relevant information.

## Death

If your health reaches 0, you will die. When dead:
- You cannot move or use most commands
- You can use the `respawn` command to return to life
- Respawning will return you to the starting area

## Tips

- Make sure your health is high before engaging in combat
- Flee if your health gets too low
- Some enemies are much stronger than others - choose your battles wisely 