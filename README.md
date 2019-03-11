<div align="center">
	<img width="500" src=".github/logo.svg" alt="pinpt-logo">
</div>

<p align="center" color="#6a737d">
	<strong>Ripsrc is a library for analyzing source code inside a Git repo</strong>
</p>

## Install

```
go get -u github.com/pinpt/ripsrc
```

## Usage

You can use the example command line implementation provided.

```
ripsrc code <gitfolder>
```

This will rip through all the commits in history order (oldest to newest), analyze each file and dump out some basic results.

### API

This repo is meant to mainly be used as a library:

```golang
results := make(chan ripsrc.BlameResult, 100)
resultsDone := make(chan bool, 1)
go func(){
	for r := range results {
		fmt.Println(r)
	}
	resultsDone <- true
}()

opts := &ripsrc.Opts{}
opts.RepoDir = filepath.Join(dir, "myrepo_dir")
opts.CheckpointsDir = ...
rip := ripsrc.New(opts)
err := rip.Code(ctx, results)
if err != nil {
	log.Fatal("error", err)
}
<-resultsDone
```

## License

All of this code is Copyright &copy; 2018-2019 by Pinpoint Software, Inc. Licensed under the MIT License
