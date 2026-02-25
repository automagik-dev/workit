package cmd

type OutputPathFlag struct {
	Path string `name:"out" aliases:"output" help:"Output file path (default: workit config dir)"`
}

type OutputPathRequiredFlag struct {
	Path string `name:"out" aliases:"output" help:"Output file path (required)"`
}

type OutputDirFlag struct {
	Dir string `name:"out-dir" aliases:"output-dir" help:"Directory to write attachments to (default: current directory)"`
}
