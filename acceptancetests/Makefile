mkfile_dir := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
p=test*.py
py3="assess_model_change_watcher.py"
test:
	TMPDIR=/tmp python -m unittest discover -vv . -p "$(p)"
	TMPDIR=/tmp python3 -m unittest discover -vv jujupy -t . -p "$(p)"
lint:
	python3 -m flake8 --builtins xrange,basestring $(py3) --exclude=repository
	flake8 --builtins xrange,basestring --exclude=$(py3),repository
cover:
	python -m coverage run --source="./" --omit "./tests/*" -m unittest discover -vv ./tests
	python -m coverage report
clean:
	find . -name '*.pyc' -delete

apt-update:
	sudo apt-get -qq update
juju-ci-tools.common_0.1.4-0_all.deb: apt-update
	find . -name '*.deb' -delete
	sudo apt-get install -y equivs
	equivs-build juju-ci-tools-common
install-deps: juju-ci-tools.common_0.1.4-0_all.deb apt-update
	sudo dpkg -i juju-ci-tools.common_0.1.4-0_all.deb || true
	sudo apt-get install -y -f
	pip install --user -r $(mkfile_dir)/requirements.txt
name=NAMEHERE
assess_file=assess_$(name).py
new-assess:
	install -m 755 template_assess.py.tmpl $(assess_file)
	sed -i -e "s/TEMPLATE/$(name)/g" $(assess_file) $(test_assess_file)
.PHONY: lint test cover clean new-assess apt-update install-deps
