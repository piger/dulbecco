#!/usr/bin/env python
# -*- coding: utf-8 -*-
import cPickle as pickle
import shutil
import os
import json
import sys


class PersistentDict(dict):
    def __init__(self, filename, *args, **kwargs):
        self.filename = filename
        dict.__init__(self, *args, **kwargs)

    def save(self):
        tmpfile = self.filename + ".tmp"

        try:
            with open(tmpfile, "wb") as fd:
                pickle.dump(dict(self), fd, 2)
        except (OSError, pickle.PickleError):
            os.remove(tmpfile)
            raise

        shutil.move(tmpfile, self.filename)

    def load(self):
        if not os.path.exists(self.filename):
            return

        with open(self.filename, "rb") as fd:
            data = pickle.load(fd)
            self.update(data)

if __name__ == '__main__':
    filename = "markov.pickle"
    pd = PersistentDict(filename)
    pd.load()

    i = 0
    for key in pd:
        jkey = json.dumps(key, separators=(',', ':'))
        for subkey in pd[key]:
            line = u"%s\n%s\n" % (jkey, subkey)
            sys.stdout.write(line.encode('utf-8'))
