# maybe more powerful
# for mac (sed for linux is different)
dir=`echo ${PWD##*/}`
grep "share-liebian" * -R | grep -v Godeps | awk -F: '{print $1}' | sort | uniq | xargs sed -i '' "s#share-liebian#$dir#g"
mv share-liebian.ini $dir.ini

