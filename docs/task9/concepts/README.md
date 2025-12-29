# Task 9 Concepts Explained

This directory contains detailed explanations of concepts used in Task 9, written for learners.

## 📚 Concepts Covered

### 1. [Startup Recovery & Persistence Boundary](./01-startup-recovery.md)

- What is startup recovery
- Why recovery matters
- The problem we're solving
- Source of truth design
- Recovery rules
- Our recovery implementation
- Startup order
- Common mistakes

### 2. [Recovery Backpressure](./02-recovery-backpressure.md)

- What is recovery backpressure
- Why backpressure during recovery
- The problem
- Our solution: exponential backoff
- Implementation details
- Common mistakes

### 3. [State Transitions in Recovery](./03-state-transitions-recovery.md)

- Why state transitions matter in recovery
- The recovery transition
- Why we need processing → pending
- How we implement it
- Common mistakes

### 4. [Source of Truth Design](./04-source-of-truth.md)

- What is source of truth
- Why store is source of truth
- Queue as delivery mechanism
- Recovery implications
- Common mistakes

## 🎯 How to Use This

These documents are designed to be read **in order** if you're new to these concepts. Each concept builds on previous ones.

**Recommended reading order:**

1. Start with [Startup Recovery](./01-startup-recovery.md) - Foundation for understanding recovery
2. Then [Source of Truth Design](./04-source-of-truth.md) - Why store is authoritative
3. Then [State Transitions](./03-state-transitions-recovery.md) - How recovery respects state machine
4. Finally [Recovery Backpressure](./02-recovery-backpressure.md) - How recovery handles queue full

Or read them as you encounter concepts in the code!

## 💡 Learning Approach

Each document:

- Explains **why** things exist (not just what they do)
- Breaks down code **line by line**
- Uses **analogies** and **mental models**
- Shows **common mistakes** to avoid
- Provides **real examples** from our codebase

## 🔗 Related Resources

- [Task 9 Summary](../summary.md) - Quick reference
- [Task 9 README](../README.md) - Complete overview
- [Task 9 Description](../description.md) - Task requirements
- [Main Learnings](../../learnings.md) - Overall project learnings

## 📝 Contributing

If you find something unclear or want to add explanations, feel free to update these documents!

