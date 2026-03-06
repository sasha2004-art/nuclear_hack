"""
Скрипт сборки мобильных библиотек из plotix_core через gomobile.

Предварительные требования:
    go install golang.org/x/mobile/cmd/gomobile@latest
    go install golang.org/x/mobile/cmd/gobind@latest
    gomobile init
    Android NDK (через Android Studio SDK Manager)

Использование:
    python build_mobile.py           # сборка Android (iOS только на macOS)
    python build_mobile.py android   # только Android
    python build_mobile.py ios       # только iOS (нужен macOS + Xcode)
"""

import subprocess
import os
import sys


def run_command(command, description):
    print(f"[*] {description}...")
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        print(f"[!] ОШИБКА: {description} провалена.")
        return False
    print(f"[+] OK: {description}")
    return True


def build():
    out_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "mobile_out")
    os.makedirs(out_dir, exist_ok=True)

    target = sys.argv[1] if len(sys.argv) > 1 else "all"

    if target in ("all", "android"):
        aar_path = os.path.join(out_dir, "plotix_core.aar")
        if not run_command(
            f'gomobile bind -v -target android -o "{aar_path}" ./plotix_core/mobile',
            "Сборка Android AAR"
        ):
            print("[!] Проверьте:")
            print("    1. gomobile установлен: go install golang.org/x/mobile/cmd/gomobile@latest")
            print("    2. gomobile в PATH: %USERPROFILE%\\go\\bin")
            print("    3. Android NDK установлен (Android Studio -> SDK Manager -> SDK Tools -> NDK)")
            print("    4. ANDROID_NDK_HOME указан (если NDK не находится автоматически)")
            sys.exit(1)

    if target in ("all", "ios"):
        if os.name == "nt":
            print("[SKIP] Сборка iOS пропущена (нужен macOS + Xcode)")
        else:
            xcf_path = os.path.join(out_dir, "PlotixCore.xcframework")
            if not run_command(
                f'gomobile bind -v -target ios -o "{xcf_path}" ./plotix_core/mobile',
                "Сборка iOS xcframework"
            ):
                print("[!] Проверьте, установлен ли Xcode и gomobile")
                sys.exit(1)

    print(f"\n[DONE] Библиотеки находятся в: {out_dir}")


if __name__ == "__main__":
    build()
