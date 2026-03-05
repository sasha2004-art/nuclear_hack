import subprocess
import os
import sys
import shutil

def run_command(command, cwd, description):
    print(f"[*] {description}...")
    result = subprocess.run(command, cwd=cwd, shell=True)
    if result.returncode != 0:
        print(f"[!] Error: {description} failed.")
        sys.exit(1)

def build():
    base_dir = os.path.dirname(os.path.abspath(__file__))
    ui_dir = os.path.join(base_dir, "plotix_ui")
    core_dir = os.path.join(base_dir, "plotix_core")
    dist_target = os.path.join(core_dir, "ui_dist")

    print("--- Building Frontend ---")
    run_command("npm install", ui_dir, "npm install")
    run_command("npm run build", ui_dir, "npm run build")

    if os.path.exists(dist_target):
        shutil.rmtree(dist_target)

    source_dist = os.path.join(ui_dir, "dist")
    if not os.path.exists(source_dist):
        print("[!] Error: dist folder not found after build.")
        sys.exit(1)

    shutil.copytree(source_dist, dist_target)
    print("[+] Frontend copied to plotix_core/ui_dist")

    print("\n--- Building Go Binary ---")
    binary_name = "plotix_app.exe"
    build_cmd = f"go build -ldflags=\"-s -w\" -o ../{binary_name} ."
    run_command(build_cmd, core_dir, "go build")

    final_path = os.path.join(base_dir, binary_name)
    size_mb = os.path.getsize(final_path) / (1024 * 1024)
    print(f"\n[DONE] Built: {binary_name} ({size_mb:.1f} MB)")
    print("[INFO] Run it and open http://localhost:8080 in browser.")

if __name__ == "__main__":
    build()
