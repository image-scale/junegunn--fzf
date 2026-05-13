# Goal

## Project
fzf — a Go project implementing a general-purpose command-line fuzzy finder.

## Description
fzf is an interactive filter program for any kind of list (files, command history, processes, etc.). It implements fuzzy matching algorithms so users can quickly type patterns with omitted characters and still get relevant results. Core capabilities include:

- High-performance fuzzy matching algorithms (greedy O(n) and optimal Smith-Waterman O(n*m))
- Multiple match modes: fuzzy, exact substring, exact boundary, prefix, suffix, equal
- Extended search syntax with AND/OR operators and negation
- ANSI escape sequence parsing to preserve colored input
- Field tokenization with configurable delimiters and nth-field selection
- Chunk-based item storage for handling millions of items
- Bitmap-based query caching for incremental search refinement
- Multi-criteria result scoring and ranking with radix sort
- K-way merge of sorted results from parallel workers
- File-backed command history with cursor navigation
- Event-driven concurrent architecture with EventBox coordination
- Input reading from stdin, shell commands, or filesystem walking
- Parallel matching across item chunks using multiple workers
- Unicode-aware character handling with ASCII fast paths
- Character normalization for accent-insensitive matching

## Scope
- ~20 production source files to implement
- ~15 test files to write
- Reproduce all core source code, tests, and configuration
