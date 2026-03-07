import subprocess
import os
import sys

def run_command(command, description):
    print(f"[*] {description}...")
    # Используем shell=True для Windows
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        print(f"[!] ОШИБКА: {description} провалена.")
        return False
    print(f"[+] OK: {description}")
    return True

def build():
    # 1. Определяем пути
    # Скрипт лежит в plotix_core/mobile/
    mobile_dir = os.path.dirname(os.path.abspath(__file__))
    # Корень проекта (где должен лежать go.mod) - это папка plotix_core/
    root_dir = os.path.abspath(os.path.join(mobile_dir, ".."))

    # Переходим в корень проекта, чтобы Go видел все пакеты (accounts, api и т.д.)
    os.chdir(root_dir)

    # Папка для результата сборки
    out_dir = os.path.join(mobile_dir, "mobile_out")
    os.makedirs(out_dir, exist_ok=True)

    # 2. Проверка папки ui_dist (нужна для //go:embed)
    ui_dist_path = os.path.join(mobile_dir, "ui_dist")
    if not os.path.exists(ui_dist_path):
        print(f"[*] Создаю отсутствующую папку {ui_dist_path}...")
        os.makedirs(ui_dist_path, exist_ok=True)
        with open(os.path.join(ui_dist_path, "placeholder.txt"), "w") as f:
            f.write("placeholder for gomobile")

    target = sys.argv[1] if len(sys.argv) > 1 else "all"

    # 3. Сборка для Android
    if target in ("all", "android"):
        aar_path = os.path.join(out_dir, "plotix_core.aar")

        # Команда gomobile bind:
        # -target=android: целевая ОС
        # -androidapi 21: фиксит ошибку NDK 26 (требует минимум API 21)
        # -o: куда положить результат
        # ./mobile: какой пакет собирать (относительно корня root_dir)
        cmd = f'gomobile bind -v -target=android -androidapi 21 -ldflags="-checklinkname=0" -o "{aar_path}" ./mobile'

        if not run_command(cmd, "Сборка Android AAR"):
            print("\n[!] КРИТИЧЕСКАЯ ОШИБКА СБОРКИ")
            print("Убедитесь, что:")
            print(f"1. Файл go.mod находится в {root_dir}")
            print(f"2. В go.mod написано 'module plotix_core'")
            print("3. В папке mobile/ нет своего отдельного go.mod")
            sys.exit(1)

    # 4. Сборка для iOS (только на Mac)
    if target in ("all", "ios"):
        if os.name == "nt":
            print("[SKIP] Сборка iOS пропущена (нужен macOS + Xcode)")
        else:
            xcf_path = os.path.join(out_dir, "PlotixCore.xcframework")
            cmd = f'gomobile bind -v -target=ios -o "{xcf_path}" ./mobile'
            if not run_command(cmd, "Сборка iOS xcframework"):
                sys.exit(1)

    print(f"\n[DONE] УСПЕХ! Библиотеки находятся в: {out_dir}")

if __name__ == "__main__":
    build()