git clone https://github.com/google/android-cuttlefish

cd android-cuttlefish
for dir in base frontend; do
  cd $dir
  debuild -i -us -uc -b -d
  cd ..
done

mkdir /packages
cp -f ./*.deb /packages/
