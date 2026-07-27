{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  nativeBuildInputs = with pkgs; [
    rustc cargo protobuf gcc pkg-config
  ];

  buildInputs = with pkgs; [
    fontconfig expat wayland libxkbcommon
    xorg.libX11 xorg.libXcursor xorg.libXi xorg.libXrandr
    libGL
  ];

  shellHook = ''
    export LD_LIBRARY_PATH="$LD_LIBRARY_PATH:${pkgs.lib.makeLibraryPath [
      pkgs.wayland
      pkgs.libxkbcommon
      pkgs.xorg.libX11
      pkgs.xorg.libXcursor
      pkgs.xorg.libXi
      pkgs.xorg.libXrandr
      pkgs.libGL
    ]}"

    export WINIT_UNIX_BACKEND=x11
    export LIBGL_ALWAYS_SOFTWARE=1
  '';
}
