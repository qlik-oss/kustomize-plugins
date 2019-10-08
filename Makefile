kustomizeVersionShort := v3.2.2

install:
	./scripts/gogetKustomize.sh ${kustomizeVersionShort}
	./scripts/build.sh
