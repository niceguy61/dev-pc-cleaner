package registry

type Command struct {
	Exe  string
	Args []string
}

type Language struct {
	Name     string
	Category string
	Commands []Command
}

func Languages() []Language {
	return []Language{
		// Top-tier general languages
		lang("Python", "General", cmd("python", "--version"), cmd("py", "-V")),
		lang("JavaScript", "Web", cmd("node", "-v")),
		lang("TypeScript", "Web", cmd("tsc", "-v")),
		lang("Java", "General", cmd("java", "-version"), cmd("javac", "-version")),
		lang("C", "Systems", cmd("cl"), cmd("gcc", "--version"), cmd("clang", "--version")),
		lang("C++", "Systems", cmd("cl"), cmd("g++", "--version"), cmd("clang++", "--version")),
		lang("C#", ".NET", cmd("dotnet", "--info"), cmd("csc")),
		lang("Visual Basic .NET", ".NET", cmd("vbc")),
		lang("Go", "Systems", cmd("go", "version")),
		lang("Rust", "Systems", cmd("rustc", "-V"), cmd("cargo", "-V")),
		lang("PHP", "Web", cmd("php", "-v")),
		lang("Ruby", "General", cmd("ruby", "-v"), cmd("gem", "-v")),
		lang("Perl", "General", cmd("perl", "-v")),
		lang("Lua", "General", cmd("lua", "-v"), cmd("luajit", "-v")),
		lang("R", "Data", cmd("R", "--version")),
		lang("Julia", "Data", cmd("julia", "--version")),
		lang("MATLAB", "Data", cmd("matlab", "-batch", "version")),
		lang("PowerShell", "Shell",
			cmd("pwsh", "-NoLogo", "-NoProfile", "-Command", "$PSVersionTable.PSVersion"),
			cmd("powershell", "-NoLogo", "-NoProfile", "-Command", "$PSVersionTable.PSVersion"),
		),
		lang("Bash", "Shell", cmd("bash", "--version")),
		lang("Kotlin", "JVM", cmd("kotlinc", "-version")),
		lang("Swift", "Apple", cmd("swift", "--version")),
		lang("Objective-C", "Apple", cmd("clang", "--version")),
		lang("Scala", "JVM", cmd("scala", "-version")),
		lang("Groovy", "JVM", cmd("groovy", "-version")),
		lang("Dart", "Mobile", cmd("dart", "--version")),
		lang("Haskell", "Functional", cmd("ghc", "--version")),
		lang("Elixir", "Functional", cmd("elixir", "--version")),
		lang("Erlang", "Functional", cmd("erl", "-version")),
		lang("Clojure", "Functional", cmd("clojure", "-Sdescribe")),
		lang("Lisp", "Functional", cmd("sbcl", "--version"), cmd("clisp", "--version")),
		lang("F#", "Functional", cmd("dotnet", "fsi", "--version"), cmd("fsharpi", "--version")),
		lang("OCaml", "Functional", cmd("ocaml", "-version")),
		lang("Zig", "Systems", cmd("zig", "version")),
		lang("Nim", "Systems", cmd("nim", "--version")),
		lang("Crystal", "Systems", cmd("crystal", "--version")),
		lang("D", "Systems", cmd("dmd", "--version"), cmd("ldc2", "--version")),
		lang("Assembly", "Systems", cmd("nasm", "-v"), cmd("yasm", "--version")),
		lang("Fortran", "Legacy", cmd("gfortran", "--version"), cmd("ifort", "-V")),
		lang("Ada", "Legacy", cmd("gnat", "--version")),
		lang("COBOL", "Legacy", cmd("cobc", "-version")),
		lang("Solidity", "Blockchain", cmd("solc", "--version")),
		lang("Prolog", "Logic", cmd("swipl", "--version")),
		lang("GDScript", "Game", cmd("godot", "--version")),
		lang("SQL", "Data", cmd("sqlite3", "--version"), cmd("psql", "--version"), cmd("mysql", "--version")),
	}
}

func lang(name, category string, cmds ...Command) Language {
	return Language{Name: name, Category: category, Commands: cmds}
}

func cmd(exe string, args ...string) Command {
	return Command{Exe: exe, Args: args}
}
