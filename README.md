# GitOps Pipeline

[Dagger](https://dagger.io/) pipeline to show the usage of following Dagger modules:

* [dagger-module-git-actions](https://github.com/puzzle/dagger-module-git-actions)
* [dagger-module-helm](https://github.com/puzzle/dagger-module-helm)

The Dagger pipeline / module is located in the [gitops](./gitops/) directory.

## usage

Basic usage guide.

The [gitops](./gitops/) directory contains a [Dagger](https://dagger.io/) module.

Check the official Dagger Module documentation: https://docs.dagger.io/features/modules

The [Dagger CLI](https://docs.dagger.io/cli) is needed.

### functions

List all functions of the module. This command is provided by the [Dagger CLI](https://docs.dagger.io/cli). 

```bash
dagger functions
```

The GitOps module is referenced locally.

## development

Basic development guide.

### setup Dagger module

Setup the Dagger module:

```bash
dagger init --sdk go --name pitc-gitops --source gitops
```

## To Do

- [ ] document functions
- [ ] Update module dependencies to new repo
- [ ] Add more tools
- [ ] Add cache mounts
- [ ] Add environment variables
- [ ] Add more examples
- [ ] Add tests
