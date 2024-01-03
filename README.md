# daggerverse GitOps Module

[Dagger](https://dagger.io/) module for [daggerverse](https://daggerverse.dev/) providing GitOps functionality.

The Dagger module is located in the [gitops](./gitops/) directory.

## usage

Basic usage guide.

The [gitops](./gitops/) directory contains a [daggerverse](https://daggerverse.dev/) [Dagger](https://dagger.io/) module.

Check the official Dagger Module documentation: https://docs.dagger.io/zenith/

The [Dagger CLI](https://docs.dagger.io/cli) is needed.

### functions

List all functions of the module. This command is provided by the [Dagger CLI](https://docs.dagger.io/cli). 

```bash
dagger functions -m ./gitops/
```

The GitOps module is referenced locally.

## development

Basic development guide.

### setup Dagger module

Setup the Dagger module.

Create the directory for the module and initialize it.

```bash
mkdir gitops/
cd gitops/

# initialize Dagger module
dagger mod init --sdk go --name pitc-gitops
```

### setup development module

Setup the outer module to be able to develop the Dagger GitOps module.

```bash
dagger mod init --sdk go --name modest
dagger mod use ./gitops
```

Generate or re-generate the Go definitions file (dagger.gen.go) for use in code completion.

```bash
dagger mod install
```

The functions of the module are available by the `dag` variable. Type `dag.` in your Go file for code completion.


Update the module:

```bash
dagger mod update
```

## To Do

- [ ] document functions
- [ ] Update module dependencies to new repo
- [ ] Add more tools
- [ ] Add cache mounts
- [ ] Add environment variables
- [ ] Add more examples
- [ ] Add tests
