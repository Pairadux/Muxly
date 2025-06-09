// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package models

type Window struct {
	Name, Cmd string
}

type SessionLayout struct {
	Windows []Window
}

type Entry struct {
	Path string
	Depth int
}
